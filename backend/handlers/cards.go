package handlers

import (
	"context"
	"net/http"

	"mtgdeckbuilder/db"
	"mtgdeckbuilder/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetMyCards returns the current user's card collection with optional filters.
func GetMyCards(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	ctx := context.Background()

	filter := bson.M{"user_id": user.ID, "quantity": bson.M{"$gt": 0}}
	if rarity := c.Query("rarity"); rarity != "" {
		filter["rarity"] = rarity
	}
	if name := c.Query("name"); name != "" {
		filter["card_name"] = bson.M{"$regex": name, "$options": "i"}
	}
	if setCode := c.Query("set"); setCode != "" {
		filter["set_code"] = setCode
	}

	cursor, err := db.Col("user_cards").Find(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch cards"})
		return
	}
	var cards []models.UserCard
	cursor.All(ctx, &cards)
	if cards == nil {
		cards = []models.UserCard{}
	}
	c.JSON(http.StatusOK, cards)
}

// RecycleCard destroys 1 copy of a card and gives the user 1 JAD.
func RecycleCard(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	ctx := context.Background()

	ucID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var uc models.UserCard
	if err := db.Col("user_cards").FindOne(ctx, bson.M{"_id": ucID, "user_id": user.ID}).Decode(&uc); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
		return
	}
	if uc.Quantity < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no copies to recycle"})
		return
	}

	db.Col("user_cards").UpdateOne(ctx, bson.M{"_id": ucID}, bson.M{"$inc": bson.M{"quantity": -1}})
	db.Col("users").UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{"$inc": bson.M{"jad": 1}})

	// return updated JAD
	var updated models.User
	db.Col("users").FindOne(ctx, bson.M{"_id": user.ID}).Decode(&updated)
	c.JSON(http.StatusOK, gin.H{"jad": updated.JAD})
}

// SearchCards searches the master card DB for autocomplete.
func SearchCards(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(http.StatusOK, []models.Card{})
		return
	}
	ctx := context.Background()
	limit := int64(20)
	cursor, err := db.Col("cards").Find(ctx,
		bson.M{"name": bson.M{"$regex": q, "$options": "i"}, "banned": false},
		&options.FindOptions{Limit: &limit},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}
	var cards []models.Card
	cursor.All(ctx, &cards)
	if cards == nil {
		cards = []models.Card{}
	}
	c.JSON(http.StatusOK, cards)
}
