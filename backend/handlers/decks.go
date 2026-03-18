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
)

func GetDecks(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	cursor, _ := db.Col("decks").Find(context.Background(), bson.M{"user_id": user.ID})
	var decks []models.Deck
	cursor.All(context.Background(), &decks)
	if decks == nil {
		decks = []models.Deck{}
	}
	c.JSON(http.StatusOK, decks)
}

func GetDeck(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var deck models.Deck
	if err := db.Col("decks").FindOne(context.Background(), bson.M{"_id": id, "user_id": user.ID}).Decode(&deck); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "deck not found"})
		return
	}
	c.JSON(http.StatusOK, deck)
}

func CreateDeck(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	var body struct {
		Name string `json:"name" binding:"required,min=1,max=60"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	deck := models.Deck{
		ID:        primitive.NewObjectID(),
		UserID:    user.ID,
		Name:      body.Name,
		Cards:     []models.DeckCard{},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	db.Col("decks").InsertOne(context.Background(), deck)
	c.JSON(http.StatusCreated, deck)
}

func UpdateDeck(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		Name  string            `json:"name"`
		Cards []models.DeckCard `json:"cards"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	update := bson.M{"$set": bson.M{
		"updated_at": time.Now().UTC(),
	}}
	if body.Name != "" {
		update["$set"].(bson.M)["name"] = body.Name
	}
	if body.Cards != nil {
		update["$set"].(bson.M)["cards"] = body.Cards
	}
	res, _ := db.Col("decks").UpdateOne(context.Background(), bson.M{"_id": id, "user_id": user.ID}, update)
	if res.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "deck not found"})
		return
	}
	var deck models.Deck
	db.Col("decks").FindOne(context.Background(), bson.M{"_id": id}).Decode(&deck)
	c.JSON(http.StatusOK, deck)
}

func DeleteDeck(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	res, _ := db.Col("decks").DeleteOne(context.Background(), bson.M{"_id": id, "user_id": user.ID})
	if res.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "deck not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// AddCardToDeck adds one copy of a card to a deck, or increments quantity if already present.
func AddCardToDeck(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		CardID   string `json:"card_id" binding:"required"`
		CardName string `json:"card_name" binding:"required"`
		ImageURI string `json:"image_uri"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := context.Background()
	var deck models.Deck
	if err := db.Col("decks").FindOne(ctx, bson.M{"_id": id, "user_id": user.ID}).Decode(&deck); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "deck not found"})
		return
	}

	cardOID, _ := primitive.ObjectIDFromHex(body.CardID)
	found := false
	for i, dc := range deck.Cards {
		if dc.CardID == cardOID {
			deck.Cards[i].Quantity++
			found = true
			break
		}
	}
	if !found {
		deck.Cards = append(deck.Cards, models.DeckCard{
			CardID:   cardOID,
			CardName: body.CardName,
			ImageURI: body.ImageURI,
			Quantity: 1,
		})
	}

	db.Col("decks").UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"cards":      deck.Cards,
		"updated_at": time.Now().UTC(),
	}})
	c.JSON(http.StatusOK, deck)
}

// GetOwnedCardCounts returns a map of card_name -> owned quantity for the current user.
func GetOwnedCardCounts(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	cursor, _ := db.Col("user_cards").Find(context.Background(), bson.M{"user_id": user.ID, "quantity": bson.M{"$gt": 0}})
	var cards []models.UserCard
	cursor.All(context.Background(), &cards)
	result := map[string]int{}
	for _, uc := range cards {
		result[uc.CardName] += uc.Quantity
	}
	c.JSON(http.StatusOK, result)
}
