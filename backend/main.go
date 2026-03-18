package main

import (
	"context"
	"log"
	"os"
	"time"

	"mtgdeckbuilder/config"
	"mtgdeckbuilder/db"
	"mtgdeckbuilder/handlers"
	"mtgdeckbuilder/middleware"
	"mtgdeckbuilder/seeds"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	middleware.JWTSecret = []byte(cfg.JWTSecret)

	db.Connect(cfg.MongoURI)

	// Seed cards in background
	go seeds.SeedCards()

	// Background job: process expired market bids every minute
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			handlers.ProcessExpiredBids(context.Background())
		}
	}()

	r := gin.New()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	api := r.Group("/api")

	// Seed status (public)
	api.GET("/seed-status", func(c *gin.Context) {
		c.JSON(200, seeds.GetStatus())
	})

	// Auth (public)
	api.POST("/auth/register", handlers.Register)
	api.POST("/auth/login", handlers.Login)

	// Protected routes
	auth := api.Group("/", middleware.Auth())
	{
		auth.GET("/me", handlers.Me)

		// Boosters
		auth.GET("/boosters", handlers.GetBoosterStatus)
		auth.POST("/boosters/open", handlers.OpenBooster)

		// Cards
		auth.GET("/cards", handlers.GetMyCards)
		auth.POST("/cards/:id/recycle", handlers.RecycleCard)
		auth.GET("/cards/search", handlers.SearchCards)
		auth.GET("/cards/owned-counts", handlers.GetOwnedCardCounts)

		// Decks
		auth.GET("/decks", handlers.GetDecks)
		auth.POST("/decks", handlers.CreateDeck)
		auth.GET("/decks/:id", handlers.GetDeck)
		auth.PUT("/decks/:id", handlers.UpdateDeck)
		auth.DELETE("/decks/:id", handlers.DeleteDeck)
		auth.POST("/decks/:id/add-card", handlers.AddCardToDeck)

		// Market
		auth.GET("/market", handlers.GetMarket)
		auth.POST("/market/:id/bid", handlers.BidOnCard)
		auth.POST("/market/:id/hate", handlers.HateCard)

		// Trade
		auth.GET("/trades", handlers.GetTrades)
		auth.POST("/trades", handlers.CreateTrade)
		auth.PUT("/trades/:id/accept", handlers.AcceptTrade)
		auth.PUT("/trades/:id/decline", handlers.DeclineTrade)
		auth.GET("/users", handlers.GetAllUsers)
		auth.GET("/users/:userId/cards", handlers.GetUserCards)

		// Admin
		admin := auth.Group("/admin", middleware.AdminOnly())
		{
			admin.GET("/users", handlers.AdminGetUsers)
			admin.PUT("/users/:id/admin", handlers.AdminSetAdmin)
			admin.GET("/sets", handlers.AdminGetSets)
			admin.PUT("/sets/:id/toggle", handlers.AdminToggleSet)
			admin.POST("/sets/format/:format", handlers.AdminApplyFormatPreset)
			admin.GET("/banlist", handlers.AdminGetBanlist)
			admin.POST("/banlist", handlers.AdminBanCard)
			admin.DELETE("/banlist/:id", handlers.AdminUnbanCard)
		}
	}

	// Serve React SPA: serve static files from /app/frontend/dist, fallback to index.html
	const distDir = "/app/frontend/dist"
	r.NoRoute(func(c *gin.Context) {
		filePath := distDir + c.Request.URL.Path
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			c.File(filePath)
		} else {
			c.File(distDir + "/index.html")
		}
	})

	log.Printf("Server running on :%s", cfg.Port)
	r.Run(":" + cfg.Port)
}
