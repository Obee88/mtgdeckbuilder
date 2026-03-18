package handlers

import (
	"context"
	"math/rand"
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

const initialBoosters = 10

// countEarnedBoosters calculates how many boosters a user has earned based on
// Mondays elapsed since registration, plus the initial bonus.
func countEarnedBoosters(registeredAt time.Time) int {
	now := time.Now().UTC()
	// Find the first Monday at or after registration
	reg := registeredAt.UTC()
	// weekday: 0=Sunday, 1=Monday ... 6=Saturday
	daysUntilMonday := (8 - int(reg.Weekday())) % 7
	if daysUntilMonday == 0 {
		daysUntilMonday = 7
	}
	firstMonday := time.Date(reg.Year(), reg.Month(), reg.Day()+daysUntilMonday, 0, 0, 0, 0, time.UTC)
	weekly := 0
	if !firstMonday.After(now) {
		weekly = int(now.Sub(firstMonday).Hours()/168) + 1
	}
	return initialBoosters + weekly
}

func nextMonday(registeredAt time.Time) time.Time {
	earned := countEarnedBoosters(registeredAt)
	reg := registeredAt.UTC()
	daysUntilMonday := (8 - int(reg.Weekday())) % 7
	if daysUntilMonday == 0 {
		daysUntilMonday = 7
	}
	firstMonday := time.Date(reg.Year(), reg.Month(), reg.Day()+daysUntilMonday, 0, 0, 0, 0, time.UTC)
	return firstMonday.Add(time.Duration(earned) * 7 * 24 * time.Hour)
}

// GetBoosterStatus returns available booster count and next booster time.
func GetBoosterStatus(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	earned := countEarnedBoosters(user.RegisteredAt)
	available := earned - user.BoostersOpened
	next := nextMonday(user.RegisteredAt)
	c.JSON(http.StatusOK, gin.H{
		"available":    available,
		"next_booster": next,
	})
}

// OpenBooster generates 30 cards and adds them to the user's collection.
func OpenBooster(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	ctx := context.Background()

	earned := countEarnedBoosters(user.RegisteredAt)
	available := earned - user.BoostersOpened
	if available <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no boosters available"})
		return
	}

	enabledSets, _ := db.Col("sets").CountDocuments(ctx, bson.M{"enabled": true})
	if enabledSets == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no sets are enabled — ask an admin to enable sets in Settings"})
		return
	}

	// Rarity distribution for 30-card booster:
	// 1 mythic (or replace with rare ~87.5% chance), 3 rare, 7 uncommon, 19 common
	raritySlots := []struct {
		rarity string
		count  int
	}{
		{"mythic", 1},
		{"rare", 3},
		{"uncommon", 7},
		{"common", 19},
	}

	// For the 1 mythic slot: 12.5% mythic, 87.5% rare
	if rand.Float64() < 0.875 {
		raritySlots[0].rarity = "rare"
		raritySlots[1].count = 4
		raritySlots[0].count = 0
	}

	cardIDs := []primitive.ObjectID{}
	userCards := []models.UserCard{}

	for _, slot := range raritySlots {
		if slot.count == 0 {
			continue
		}
		// pick random cards from enabled, non-banned sets
		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: bson.M{"rarity": slot.rarity, "set_enabled": true, "banned": false, "type_line": bson.M{"$not": bson.M{"$regex": "^Basic Land"}}}}},
			{{Key: "$sample", Value: bson.M{"size": slot.count}}},
		}
		cursor, err := db.Col("cards").Aggregate(ctx, pipeline)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to draw cards"})
			return
		}
		var drawn []models.Card
		cursor.All(ctx, &drawn)

		for _, card := range drawn {
			// upsert user card (increment quantity)
			filter := bson.M{"user_id": user.ID, "card_id": card.ID}
			update := bson.M{
				"$inc": bson.M{"quantity": 1},
				"$setOnInsert": bson.M{
					"_id":       primitive.NewObjectID(),
					"user_id":   user.ID,
					"card_id":   card.ID,
					"card_name": card.Name,
					"set_code":  card.SetCode,
					"set_name":  card.SetName,
					"rarity":    card.Rarity,
					"image_uri": card.ImageURI,
					"mana_cost": card.ManaCost,
					"type_line": card.TypeLine,
				},
			}
			opts := options.Update().SetUpsert(true)
			res, _ := db.Col("user_cards").UpdateOne(ctx, filter, update, opts)
			var ucID primitive.ObjectID
			if res.UpsertedID != nil {
				ucID = res.UpsertedID.(primitive.ObjectID)
			} else {
				var uc models.UserCard
				db.Col("user_cards").FindOne(ctx, filter).Decode(&uc)
				ucID = uc.ID
			}
			cardIDs = append(cardIDs, ucID)

			uc := models.UserCard{
				ID:       ucID,
				CardID:   card.ID,
				CardName: card.Name,
				Rarity:   card.Rarity,
				ImageURI: card.ImageURI,
				ManaCost: card.ManaCost,
				TypeLine: card.TypeLine,
				SetCode:  card.SetCode,
				SetName:  card.SetName,
			}
			userCards = append(userCards, uc)
		}
	}

	// record booster history
	history := models.BoosterHistory{
		ID:       primitive.NewObjectID(),
		UserID:   user.ID,
		OpenedAt: time.Now().UTC(),
		Cards:    cardIDs,
	}
	db.Col("booster_history").InsertOne(ctx, history)

	// increment boosters_opened
	db.Col("users").UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{"$inc": bson.M{"boosters_opened": 1}})

	c.JSON(http.StatusOK, gin.H{"cards": userCards})
}
