package handlers

import (
	"context"
	"net/http"
	"time"

	"mtgdeckbuilder/db"
	"mtgdeckbuilder/middleware"
	"mtgdeckbuilder/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

func Register(c *gin.Context) {
	var body struct {
		Username string `json:"username" binding:"required,min=3,max=30"`
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	// check duplicate
	count, _ := db.Col("users").CountDocuments(ctx, bson.M{"username": body.Username})
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "username already taken"})
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)

	// first ever user becomes admin
	totalUsers, _ := db.Col("users").CountDocuments(ctx, bson.M{})
	isAdmin := totalUsers == 0

	user := models.User{
		ID:           primitive.NewObjectID(),
		Username:     body.Username,
		Password:     string(hash),
		IsAdmin:      isAdmin,
		JAD:          0,
		JADLocked:    0,
		RegisteredAt: time.Now().UTC(),
		BoostersOpened: 0,
	}

	if _, err := db.Col("users").InsertOne(ctx, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	token := generateToken(user)
	c.JSON(http.StatusCreated, gin.H{"token": token, "user": user})
}

func Login(c *gin.Context) {
	var body struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := db.Col("users").FindOne(context.Background(), bson.M{"username": body.Username}).Decode(&user); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token := generateToken(user)
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

func Me(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	c.JSON(http.StatusOK, user)
}

func generateToken(user models.User) string {
	claims := jwt.MapClaims{
		"user_id":  user.ID.Hex(),
		"username": user.Username,
		"is_admin": user.IsAdmin,
		"exp":      time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString(middleware.JWTSecret)
	return signed
}
