package handlers

import (
	"context"
	"math"
	"net/http"
	"time"

	"mtgdeckbuilder/db"
	"mtgdeckbuilder/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const marketSize = 20
const startingBid = 20
const bidRaise = 1.10    // +10%
const hateDiscount = 0.85 // -15%
const hatesToBan = 4
const bidWindow = 24 * time.Hour

// GetMarket returns all active market cards.
func GetMarket(c *gin.Context) {
	ctx := context.Background()
	ensureMarketFull(ctx)

	cursor, _ := db.Col("market").Find(ctx, bson.M{"status": "active"})
	var cards []models.MarketCard
	cursor.All(ctx, &cards)
	if cards == nil {
		cards = []models.MarketCard{}
	}
	c.JSON(http.StatusOK, cards)
}

// BidOnCard places a bid on a market card.
func BidOnCard(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	ctx := context.Background()

	marketID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var mc models.MarketCard
	if err := db.Col("market").FindOne(ctx, bson.M{"_id": marketID, "status": "active"}).Decode(&mc); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "market card not found"})
		return
	}

	// Can't bid on your own current bid
	if mc.CurrentBidderID != nil && *mc.CurrentBidderID == user.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "you are already the highest bidder"})
		return
	}

	bidRequired := mc.CurrentBid
	if mc.CurrentBidderID != nil {
		// raise by 10%
		bidRequired = int(math.Ceil(float64(mc.CurrentBid) * bidRaise))
	}

	// Check JAD availability (jad - jad_locked >= bidRequired)
	available := user.JAD - user.JADLocked
	if available < bidRequired {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient JAD"})
		return
	}

	// Unlock previous bidder's JAD
	if mc.CurrentBidderID != nil {
		db.Col("users").UpdateOne(ctx,
			bson.M{"_id": mc.CurrentBidderID},
			bson.M{"$inc": bson.M{"jad_locked": -mc.CurrentBid}},
		)
	}

	// Lock this user's JAD
	db.Col("users").UpdateOne(ctx,
		bson.M{"_id": user.ID},
		bson.M{"$inc": bson.M{"jad_locked": bidRequired}},
	)

	expiry := time.Now().UTC().Add(bidWindow)
	db.Col("market").UpdateOne(ctx, bson.M{"_id": marketID}, bson.M{"$set": bson.M{
		"current_bid":        bidRequired,
		"current_bidder_id":  user.ID,
		"current_bidder":     user.Username,
		"bid_expires_at":     expiry,
	}})

	c.JSON(http.StatusOK, gin.H{"bid": bidRequired, "expires_at": expiry})
}

// HateCard registers a hate vote on a market card.
func HateCard(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	ctx := context.Background()

	marketID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var mc models.MarketCard
	if err := db.Col("market").FindOne(ctx, bson.M{"_id": marketID, "status": "active"}).Decode(&mc); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "market card not found"})
		return
	}

	// Can't hate a card that has been bid on
	if mc.CurrentBidderID != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot hate a card with bids"})
		return
	}

	// Check already hated
	for _, id := range mc.HaterIDs {
		if id == user.ID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "already hated"})
			return
		}
	}

	newHateCount := mc.HateCount + 1
	newBid := int(math.Max(1, math.Floor(float64(mc.CurrentBid)*hateDiscount)))

	update := bson.M{
		"$push": bson.M{"hater_ids": user.ID},
		"$inc":  bson.M{"hate_count": 1},
		"$set":  bson.M{"current_bid": newBid},
	}
	db.Col("market").UpdateOne(ctx, bson.M{"_id": marketID}, update)

	// If 4 hates: ban the card name
	if newHateCount >= hatesToBan {
		banCardByName(ctx, mc.CardName)
		// remove from market
		db.Col("market").UpdateOne(ctx, bson.M{"_id": marketID}, bson.M{"$set": bson.M{"status": "banned"}})
		ensureMarketFull(ctx)
	}

	c.JSON(http.StatusOK, gin.H{"hate_count": newHateCount, "current_bid": newBid})
}

// ProcessExpiredBids should be called periodically. Awards won cards.
func ProcessExpiredBids(ctx context.Context) {
	now := time.Now().UTC()
	cursor, err := db.Col("market").Find(ctx, bson.M{
		"status":         "active",
		"bid_expires_at": bson.M{"$lte": now},
		"current_bidder_id": bson.M{"$exists": true, "$ne": nil},
	})
	if err != nil {
		return
	}
	var won []models.MarketCard
	cursor.All(ctx, &won)

	for _, mc := range won {
		// deduct JAD from winner
		db.Col("users").UpdateOne(ctx,
			bson.M{"_id": mc.CurrentBidderID},
			bson.M{"$inc": bson.M{
				"jad":        -mc.CurrentBid,
				"jad_locked": -mc.CurrentBid,
			}},
		)

		// give card to winner
		filter := bson.M{"user_id": mc.CurrentBidderID, "card_id": mc.CardID}
		update := bson.M{
			"$inc": bson.M{"quantity": 1},
			"$setOnInsert": bson.M{
				"_id":       primitive.NewObjectID(),
				"user_id":   mc.CurrentBidderID,
				"card_id":   mc.CardID,
				"card_name": mc.CardName,
				"rarity":    mc.Rarity,
				"image_uri": mc.ImageURI,
				"mana_cost": mc.ManaCost,
				"type_line": mc.TypeLine,
			},
		}
		opts := options.Update().SetUpsert(true)
		db.Col("user_cards").UpdateOne(ctx, filter, update, opts)

		// mark card as won
		db.Col("market").UpdateOne(ctx, bson.M{"_id": mc.ID}, bson.M{"$set": bson.M{"status": "won"}})
	}

	if len(won) > 0 {
		ensureMarketFull(ctx)
	}
}

// ensureMarketFull fills the market up to 20 active cards.
func ensureMarketFull(ctx context.Context) {
	count, _ := db.Col("market").CountDocuments(ctx, bson.M{"status": "active"})
	needed := int64(marketSize) - count
	if needed <= 0 {
		return
	}

	// Get card IDs already in market
	cursor, _ := db.Col("market").Find(ctx, bson.M{"status": "active"})
	var existing []models.MarketCard
	cursor.All(ctx, &existing)
	excludeIDs := make([]primitive.ObjectID, len(existing))
	for i, mc := range existing {
		excludeIDs[i] = mc.CardID
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"set_enabled": true,
			"banned":      false,
			"_id":         bson.M{"$nin": excludeIDs},
			"type_line":   bson.M{"$not": bson.M{"$regex": "^Basic Land"}},
		}}},
		{{Key: "$sample", Value: bson.M{"size": needed}}},
	}
	cardCursor, err := db.Col("cards").Aggregate(ctx, pipeline)
	if err != nil {
		return
	}
	var newCards []models.Card
	cardCursor.All(ctx, &newCards)

	for _, card := range newCards {
		mc := models.MarketCard{
			ID:        primitive.NewObjectID(),
			CardID:    card.ID,
			CardName:  card.Name,
			ImageURI:  card.ImageURI,
			Rarity:    card.Rarity,
			ManaCost:  card.ManaCost,
			TypeLine:  card.TypeLine,
			CurrentBid: startingBid,
			HaterIDs:  []primitive.ObjectID{},
			HateCount: 0,
			Status:    "active",
			AddedAt:   time.Now().UTC(),
		}
		db.Col("market").InsertOne(ctx, mc)
	}
}

func banCardByName(ctx context.Context, cardName string) {
	// Check not already banned
	count, _ := db.Col("banned_cards").CountDocuments(ctx, bson.M{"card_name": cardName})
	if count > 0 {
		return
	}
	ban := models.BannedCard{
		ID:       primitive.NewObjectID(),
		CardName: cardName,
		Reason:   "hates",
		BannedAt: time.Now().UTC(),
	}
	db.Col("banned_cards").InsertOne(ctx, ban)
	// Mark all cards with this name as banned
	db.Col("cards").UpdateMany(ctx, bson.M{"name": cardName}, bson.M{"$set": bson.M{"banned": true}})
}
