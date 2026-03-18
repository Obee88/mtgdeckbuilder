package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"mtgdeckbuilder/db"
	"mtgdeckbuilder/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ─── Users ───────────────────────────────────────────────────────────────────

func AdminGetUsers(c *gin.Context) {
	cursor, _ := db.Col("users").Find(context.Background(), bson.M{})
	var users []models.User
	cursor.All(context.Background(), &users)
	if users == nil {
		users = []models.User{}
	}
	c.JSON(http.StatusOK, users)
}

func AdminSetAdmin(c *gin.Context) {
	targetID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		IsAdmin bool `json:"is_admin"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	db.Col("users").UpdateOne(context.Background(), bson.M{"_id": targetID},
		bson.M{"$set": bson.M{"is_admin": body.IsAdmin}})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ─── Sets ────────────────────────────────────────────────────────────────────

func AdminGetSets(c *gin.Context) {
	cursor, _ := db.Col("sets").Find(context.Background(), bson.M{})
	var sets []models.MTGSet
	cursor.All(context.Background(), &sets)
	if sets == nil {
		sets = []models.MTGSet{}
	}
	c.JSON(http.StatusOK, sets)
}

func AdminToggleSet(c *gin.Context) {
	setID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := context.Background()

	var set models.MTGSet
	if err := db.Col("sets").FindOne(ctx, bson.M{"_id": setID}).Decode(&set); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "set not found"})
		return
	}

	db.Col("sets").UpdateOne(ctx, bson.M{"_id": setID}, bson.M{"$set": bson.M{"enabled": body.Enabled}})
	// Sync cards for this set
	db.Col("cards").UpdateMany(ctx, bson.M{"set_code": set.Code}, bson.M{"$set": bson.M{"set_enabled": body.Enabled}})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ─── Banlist ──────────────────────────────────────────────────────────────────

func AdminGetBanlist(c *gin.Context) {
	cursor, _ := db.Col("banned_cards").Find(context.Background(), bson.M{})
	var bans []models.BannedCard
	cursor.All(context.Background(), &bans)
	if bans == nil {
		bans = []models.BannedCard{}
	}
	c.JSON(http.StatusOK, bans)
}

func AdminBanCard(c *gin.Context) {
	var body struct {
		CardName string `json:"card_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := context.Background()
	count, _ := db.Col("banned_cards").CountDocuments(ctx, bson.M{"card_name": body.CardName})
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "already banned"})
		return
	}
	ban := models.BannedCard{
		ID:       primitive.NewObjectID(),
		CardName: body.CardName,
		Reason:   "manual",
		BannedAt: time.Now().UTC(),
	}
	db.Col("banned_cards").InsertOne(ctx, ban)
	db.Col("cards").UpdateMany(ctx, bson.M{"name": body.CardName}, bson.M{"$set": bson.M{"banned": true}})
	c.JSON(http.StatusCreated, ban)
}

func AdminUnbanCard(c *gin.Context) {
	banID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := context.Background()
	var ban models.BannedCard
	if err := db.Col("banned_cards").FindOne(ctx, bson.M{"_id": banID}).Decode(&ban); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ban not found"})
		return
	}
	db.Col("banned_cards").DeleteOne(ctx, bson.M{"_id": banID})
	db.Col("cards").UpdateMany(ctx, bson.M{"name": ban.CardName}, bson.M{"$set": bson.M{"banned": false}})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ─── Format presets ───────────────────────────────────────────────────────────

type scryfallSetEntry struct {
	Code        string `json:"code"`
	SetType     string `json:"set_type"`
	ReleasedAt  string `json:"released_at"`
	Digital     bool   `json:"digital"`
}

type scryfallSetsResponse struct {
	Data    []scryfallSetEntry `json:"data"`
	HasMore bool               `json:"has_more"`
	NextPage string            `json:"next_page"`
}

func fetchScryfallSets() ([]scryfallSetEntry, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	url := "https://api.scryfall.com/sets"
	var all []scryfallSetEntry
	for url != "" {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "MtgDeckBuilder/1.0")
		req.Header.Set("Accept", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			return nil, fmt.Errorf("scryfall /sets returned HTTP %d", resp.StatusCode)
		}
		var page scryfallSetsResponse
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode scryfall sets: %w", err)
		}
		resp.Body.Close()
		all = append(all, page.Data...)
		if page.HasMore && page.NextPage != "" {
			url = page.NextPage
		} else {
			url = ""
		}
	}
	return all, nil
}

// AdminApplyFormatPreset enables only sets legal in "standard" or "modern".
func AdminApplyFormatPreset(c *gin.Context) {
	format := c.Param("format")
	if format != "standard" && format != "modern" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format must be standard or modern"})
		return
	}

	sets, err := fetchScryfallSets()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch sets from Scryfall"})
		return
	}

	// Set types that appear in boosters / are draftable
	playableTypes := map[string]bool{"expansion": true, "core": true}

	// Modern cutoff: 8th Edition released 2003-07-28
	const modernCutoff = "2003-07-28"
	// Standard: roughly last 3 years (conservative — covers all current standard sets)
	standardCutoff := time.Now().AddDate(-3, 0, 0).Format("2006-01-02")

	enabledCodes := map[string]bool{}
	for _, s := range sets {
		if s.Digital || !playableTypes[s.SetType] {
			continue
		}
		switch format {
		case "modern":
			if s.ReleasedAt >= modernCutoff {
				enabledCodes[s.Code] = true
			}
		case "standard":
			if s.ReleasedAt >= standardCutoff {
				enabledCodes[s.Code] = true
			}
		}
	}

	if len(enabledCodes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("no %s sets found in Scryfall catalog", format)})
		return
	}

	ctx := context.Background()
	// Disable all sets first, then enable matching ones
	db.Col("sets").UpdateMany(ctx, bson.M{}, bson.M{"$set": bson.M{"enabled": false}})
	db.Col("cards").UpdateMany(ctx, bson.M{}, bson.M{"$set": bson.M{"set_enabled": false}})

	codes := make([]string, 0, len(enabledCodes))
	for code := range enabledCodes {
		codes = append(codes, code)
	}
	db.Col("sets").UpdateMany(ctx, bson.M{"code": bson.M{"$in": codes}}, bson.M{"$set": bson.M{"enabled": true}})
	db.Col("cards").UpdateMany(ctx, bson.M{"set_code": bson.M{"$in": codes}}, bson.M{"$set": bson.M{"set_enabled": true}})

	enabled, _ := db.Col("sets").CountDocuments(ctx, bson.M{"enabled": true})
	c.JSON(http.StatusOK, gin.H{"enabled_sets": enabled})
}
