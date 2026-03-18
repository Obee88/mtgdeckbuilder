package handlers

import (
	"context"
	"net/http"
	"time"

	"mtgdeckbuilder/db"
	"mtgdeckbuilder/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetTrades returns all trades involving the current user.
func GetTrades(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	ctx := context.Background()

	filter := bson.M{"$or": []bson.M{
		{"from_user_id": user.ID},
		{"to_user_id": user.ID},
	}}
	cursor, _ := db.Col("trades").Find(ctx, filter)
	var trades []models.Trade
	cursor.All(ctx, &trades)
	if trades == nil {
		trades = []models.Trade{}
	}
	c.JSON(http.StatusOK, trades)
}

// GetAllUsers returns all users (for trade target selection).
func GetAllUsers(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	ctx := context.Background()

	cursor, _ := db.Col("users").Find(ctx, bson.M{"_id": bson.M{"$ne": user.ID}})
	var users []models.User
	cursor.All(ctx, &users)
	result := []gin.H{}
	for _, u := range users {
		result = append(result, gin.H{"id": u.ID, "username": u.Username})
	}
	c.JSON(http.StatusOK, result)
}

// GetUserCards returns another user's card collection (for trade building).
func GetUserCards(c *gin.Context) {
	targetID, err := primitive.ObjectIDFromHex(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	cursor, _ := db.Col("user_cards").Find(context.Background(),
		bson.M{"user_id": targetID, "quantity": bson.M{"$gt": 0}})
	var cards []models.UserCard
	cursor.All(context.Background(), &cards)
	if cards == nil {
		cards = []models.UserCard{}
	}
	c.JSON(http.StatusOK, cards)
}

// CreateTrade creates a new trade offer.
func CreateTrade(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	ctx := context.Background()

	var body struct {
		ToUserID       string             `json:"to_user_id" binding:"required"`
		OfferedCards   []models.TradeCard `json:"offered_cards"`
		OfferedJAD     int                `json:"offered_jad"`
		RequestedCards []models.TradeCard `json:"requested_cards"`
		RequestedJAD   int                `json:"requested_jad"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(body.OfferedCards) > 10 || len(body.RequestedCards) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "max 10 cards per side"})
		return
	}
	if len(body.OfferedCards) == 0 && body.OfferedJAD == 0 && len(body.RequestedCards) == 0 && body.RequestedJAD == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trade must include something"})
		return
	}

	// Validate offered JAD
	if body.OfferedJAD < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JAD amount"})
		return
	}
	if body.OfferedJAD > 0 {
		available := user.JAD - user.JADLocked
		if available < body.OfferedJAD {
			c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient JAD"})
			return
		}
	}

	toID, err := primitive.ObjectIDFromHex(body.ToUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to_user_id"})
		return
	}
	var toUser models.User
	if err := db.Col("users").FindOne(ctx, bson.M{"_id": toID}).Decode(&toUser); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "target user not found"})
		return
	}

	// Lock offered JAD
	if body.OfferedJAD > 0 {
		db.Col("users").UpdateOne(ctx, bson.M{"_id": user.ID},
			bson.M{"$inc": bson.M{"jad_locked": body.OfferedJAD}})
	}

	if body.OfferedCards == nil {
		body.OfferedCards = []models.TradeCard{}
	}
	if body.RequestedCards == nil {
		body.RequestedCards = []models.TradeCard{}
	}

	trade := models.Trade{
		ID:             primitive.NewObjectID(),
		FromUserID:     user.ID,
		FromUsername:   user.Username,
		ToUserID:       toID,
		ToUsername:     toUser.Username,
		OfferedCards:   body.OfferedCards,
		OfferedJAD:     body.OfferedJAD,
		RequestedCards: body.RequestedCards,
		RequestedJAD:   body.RequestedJAD,
		Status:         "pending",
		CreatedAt:      time.Now().UTC(),
	}
	db.Col("trades").InsertOne(ctx, trade)
	c.JSON(http.StatusCreated, trade)
}

// AcceptTrade executes a pending trade.
func AcceptTrade(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	ctx := context.Background()

	tradeID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var trade models.Trade
	if err := db.Col("trades").FindOne(ctx, bson.M{"_id": tradeID, "to_user_id": user.ID, "status": "pending"}).Decode(&trade); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trade not found"})
		return
	}

	// Validate receiver has enough JAD if requested
	if trade.RequestedJAD > 0 {
		available := user.JAD - user.JADLocked
		if available < trade.RequestedJAD {
			c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient JAD to accept"})
			return
		}
	}

	// Transfer JAD
	if trade.OfferedJAD > 0 {
		db.Col("users").UpdateOne(ctx, bson.M{"_id": trade.FromUserID},
			bson.M{"$inc": bson.M{"jad": -trade.OfferedJAD, "jad_locked": -trade.OfferedJAD}})
		db.Col("users").UpdateOne(ctx, bson.M{"_id": trade.ToUserID},
			bson.M{"$inc": bson.M{"jad": trade.OfferedJAD}})
	}
	if trade.RequestedJAD > 0 {
		db.Col("users").UpdateOne(ctx, bson.M{"_id": trade.ToUserID},
			bson.M{"$inc": bson.M{"jad": -trade.RequestedJAD}})
		db.Col("users").UpdateOne(ctx, bson.M{"_id": trade.FromUserID},
			bson.M{"$inc": bson.M{"jad": trade.RequestedJAD}})
	}

	// Transfer offered cards: decrement from sender, increment for receiver
	for _, tc := range trade.OfferedCards {
		db.Col("user_cards").UpdateOne(ctx,
			bson.M{"_id": tc.UserCardID, "user_id": trade.FromUserID},
			bson.M{"$inc": bson.M{"quantity": -tc.Quantity}})

		var card models.UserCard
		db.Col("user_cards").FindOne(ctx, bson.M{"_id": tc.UserCardID}).Decode(&card)

		filter := bson.M{"user_id": trade.ToUserID, "card_id": card.CardID}
		update := bson.M{
			"$inc": bson.M{"quantity": tc.Quantity},
			"$setOnInsert": bson.M{
				"_id":       primitive.NewObjectID(),
				"user_id":   trade.ToUserID,
				"card_id":   card.CardID,
				"card_name": card.CardName,
				"set_code":  card.SetCode,
				"set_name":  card.SetName,
				"rarity":    card.Rarity,
				"image_uri": card.ImageURI,
				"mana_cost": card.ManaCost,
				"type_line": card.TypeLine,
			},
		}
		db.Col("user_cards").UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	}

	// Transfer requested cards: decrement from receiver, increment for sender
	for _, tc := range trade.RequestedCards {
		db.Col("user_cards").UpdateOne(ctx,
			bson.M{"_id": tc.UserCardID, "user_id": trade.ToUserID},
			bson.M{"$inc": bson.M{"quantity": -tc.Quantity}})

		var card models.UserCard
		db.Col("user_cards").FindOne(ctx, bson.M{"_id": tc.UserCardID}).Decode(&card)

		filter := bson.M{"user_id": trade.FromUserID, "card_id": card.CardID}
		update := bson.M{
			"$inc": bson.M{"quantity": tc.Quantity},
			"$setOnInsert": bson.M{
				"_id":       primitive.NewObjectID(),
				"user_id":   trade.FromUserID,
				"card_id":   card.CardID,
				"card_name": card.CardName,
				"set_code":  card.SetCode,
				"set_name":  card.SetName,
				"rarity":    card.Rarity,
				"image_uri": card.ImageURI,
				"mana_cost": card.ManaCost,
				"type_line": card.TypeLine,
			},
		}
		db.Col("user_cards").UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	}

	db.Col("trades").UpdateOne(ctx, bson.M{"_id": tradeID}, bson.M{"$set": bson.M{"status": "accepted"}})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DeclineTrade declines a trade offer.
func DeclineTrade(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	ctx := context.Background()

	tradeID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var trade models.Trade
	if err := db.Col("trades").FindOne(ctx, bson.M{
		"_id":    tradeID,
		"status": "pending",
		"$or": []bson.M{
			{"to_user_id": user.ID},
			{"from_user_id": user.ID},
		},
	}).Decode(&trade); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trade not found"})
		return
	}

	// Unlock offered JAD
	if trade.OfferedJAD > 0 {
		db.Col("users").UpdateOne(ctx, bson.M{"_id": trade.FromUserID},
			bson.M{"$inc": bson.M{"jad_locked": -trade.OfferedJAD}})
	}

	status := "declined"
	if trade.FromUserID == user.ID {
		status = "cancelled"
	}
	db.Col("trades").UpdateOne(ctx, bson.M{"_id": tradeID}, bson.M{"$set": bson.M{"status": status}})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
