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

func GetLists(c *gin.Context) {
	userID := c.MustGet("user_id").(primitive.ObjectID)
	cursor, _ := db.Col("card_lists").Find(context.Background(), bson.M{"user_id": userID})
	var lists []models.CardList
	cursor.All(context.Background(), &lists)
	if lists == nil {
		lists = []models.CardList{}
	}
	c.JSON(http.StatusOK, lists)
}

func CreateList(c *gin.Context) {
	userID := c.MustGet("user_id").(primitive.ObjectID)
	var body struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	list := models.CardList{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		Name:      body.Name,
		CardNames: []string{},
		CreatedAt: time.Now().UTC(),
	}
	db.Col("card_lists").InsertOne(context.Background(), list)
	c.JSON(http.StatusCreated, list)
}

func DeleteList(c *gin.Context) {
	userID := c.MustGet("user_id").(primitive.ObjectID)
	listID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	res, _ := db.Col("card_lists").DeleteOne(context.Background(), bson.M{"_id": listID, "user_id": userID})
	if res.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "list not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func AddCardToList(c *gin.Context) {
	userID := c.MustGet("user_id").(primitive.ObjectID)
	listID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		CardName string `json:"card_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := context.Background()
	var list models.CardList
	if err := db.Col("card_lists").FindOne(ctx, bson.M{"_id": listID, "user_id": userID}).Decode(&list); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "list not found"})
		return
	}
	// Avoid duplicates
	for _, name := range list.CardNames {
		if name == body.CardName {
			c.JSON(http.StatusOK, list)
			return
		}
	}
	db.Col("card_lists").UpdateOne(ctx, bson.M{"_id": listID},
		bson.M{"$push": bson.M{"card_names": body.CardName}})
	list.CardNames = append(list.CardNames, body.CardName)
	c.JSON(http.StatusOK, list)
}

func RemoveCardFromList(c *gin.Context) {
	userID := c.MustGet("user_id").(primitive.ObjectID)
	listID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		CardName string `json:"card_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := context.Background()
	var list models.CardList
	if err := db.Col("card_lists").FindOne(ctx, bson.M{"_id": listID, "user_id": userID}).Decode(&list); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "list not found"})
		return
	}
	db.Col("card_lists").UpdateOne(ctx, bson.M{"_id": listID},
		bson.M{"$pull": bson.M{"card_names": body.CardName}})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
