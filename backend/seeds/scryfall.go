package seeds

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"mtgdeckbuilder/db"
	"mtgdeckbuilder/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const scryfallBulkCatalog = "https://api.scryfall.com/bulk-data"

// Status is exported so the handler can expose it.
type SeedStatus struct {
	Done    bool   `json:"done"`
	Message string `json:"message"`
}

var (
	statusMu sync.RWMutex
	status   = SeedStatus{Done: false, Message: "Checking…"}
)

func GetStatus() SeedStatus {
	statusMu.RLock()
	defer statusMu.RUnlock()
	return status
}

func setStatus(done bool, msg string) {
	statusMu.Lock()
	status = SeedStatus{Done: done, Message: msg}
	statusMu.Unlock()
	log.Println("[seeder]", msg)
}

type scryfallCard struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Set           string            `json:"set"`
	SetName       string            `json:"set_name"`
	Rarity        string            `json:"rarity"`
	ManaCost      string            `json:"mana_cost"`
	TypeLine      string            `json:"type_line"`
	OracleText    string            `json:"oracle_text"`
	ColorIdentity []string          `json:"color_identity"`
	ImageURIs     map[string]string `json:"image_uris"`
	Lang          string            `json:"lang"`
	Layout        string            `json:"layout"`
	Digital       bool              `json:"digital"`
}

type bulkDataResponse struct {
	Data []struct {
		Type        string `json:"type"`
		DownloadURI string `json:"download_uri"`
	} `json:"data"`
}

// SeedCards downloads Scryfall bulk data and seeds the cards collection.
// Skips if collection already has data.
func SeedCards() {
	ctx := context.Background()
	count, _ := db.Col("cards").CountDocuments(ctx, bson.M{})
	if count > 0 {
		setStatus(true, fmt.Sprintf("Ready — %d cards loaded", count))
		return
	}

	setStatus(false, "Fetching Scryfall bulk data catalog…")
	downloadURL, err := getBulkDownloadURL()
	if err != nil {
		setStatus(false, fmt.Sprintf("Failed to fetch catalog: %v", err))
		return
	}
	if downloadURL == "" {
		setStatus(false, "Could not find default_cards in Scryfall catalog")
		return
	}

	setStatus(false, "Downloading card data from Scryfall (~250 MB, please wait)…")
	cards, err := downloadAndParse(downloadURL)
	if err != nil {
		setStatus(false, fmt.Sprintf("Download/parse failed: %v", err))
		return
	}

	setStatus(false, fmt.Sprintf("Inserting %d cards into MongoDB…", len(cards)))
	insertCards(ctx, cards)

	finalCount, _ := db.Col("cards").CountDocuments(ctx, bson.M{})
	setStatus(true, fmt.Sprintf("Ready — %d cards loaded", finalCount))
}

func scryfallGet(url string, timeout time.Duration) (*http.Response, error) {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "MtgDeckBuilder/1.0")
	req.Header.Set("Accept", "application/json")
	return client.Do(req)
}

func getBulkDownloadURL() (string, error) {
	resp, err := scryfallGet(scryfallBulkCatalog, 30*time.Second)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("catalog returned HTTP %d", resp.StatusCode)
	}

	var catalog bulkDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&catalog); err != nil {
		return "", err
	}

	types := make([]string, 0, len(catalog.Data))
	for _, item := range catalog.Data {
		types = append(types, item.Type)
	}
	log.Printf("[seeder] Scryfall catalog types: %v", types)

	// Prefer default_cards, fall back to oracle_cards
	for _, want := range []string{"default_cards", "oracle_cards"} {
		for _, item := range catalog.Data {
			if item.Type == want {
				log.Printf("[seeder] Using bulk type: %s -> %s", item.Type, item.DownloadURI)
				return item.DownloadURI, nil
			}
		}
	}
	return "", fmt.Errorf("no usable bulk data type found in catalog (got: %v)", types)
}

func downloadAndParse(url string) ([]scryfallCard, error) {
	resp, err := scryfallGet(url, 10*time.Minute)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer gr.Close()
		reader = gr
	}

	var cards []scryfallCard
	if err := json.NewDecoder(reader).Decode(&cards); err != nil {
		return nil, err
	}
	return cards, nil
}

func insertCards(ctx context.Context, scryfallCards []scryfallCard) {
	db.Col("cards").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "name", Value: 1}}},
		{Keys: bson.D{{Key: "set_code", Value: 1}}},
		{Keys: bson.D{{Key: "rarity", Value: 1}}},
		{Keys: bson.D{{Key: "set_enabled", Value: 1}}},
		{Keys: bson.D{{Key: "banned", Value: 1}}},
		{Keys: bson.D{{Key: "scryfall_id", Value: 1}}, Options: options.Index().SetUnique(true)},
	})

	setMap := map[string]string{}
	var cardDocs []interface{}
	seen := map[string]bool{}

	for _, sc := range scryfallCards {
		if sc.Lang != "en" || sc.Digital {
			continue
		}
		if sc.Layout == "token" || sc.Layout == "double_faced_token" || sc.Layout == "emblem" || sc.Layout == "art_series" {
			continue
		}
		if _, ok := sc.ImageURIs["normal"]; !ok {
			continue
		}
		if seen[sc.ID] {
			continue
		}
		seen[sc.ID] = true

		setMap[sc.Set] = sc.SetName

		card := models.Card{
			ID:            primitive.NewObjectID(),
			ScryfallID:    sc.ID,
			Name:          sc.Name,
			SetCode:       sc.Set,
			SetName:       sc.SetName,
			Rarity:        sc.Rarity,
			ImageURI:      sc.ImageURIs["normal"],
			ManaCost:      sc.ManaCost,
			TypeLine:      sc.TypeLine,
			OracleText:    sc.OracleText,
			ColorIdentity: sc.ColorIdentity,
			SetEnabled:    true,
			Banned:        false,
		}
		cardDocs = append(cardDocs, card)
	}

	batchSize := 500
	for i := 0; i < len(cardDocs); i += batchSize {
		end := i + batchSize
		if end > len(cardDocs) {
			end = len(cardDocs)
		}
		db.Col("cards").InsertMany(ctx, cardDocs[i:end])
	}

	// Seed sets — all enabled by default
	db.Col("sets").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "code", Value: 1}}, Options: options.Index().SetUnique(true),
	})
	for code, name := range setMap {
		set := models.MTGSet{
			ID:      primitive.NewObjectID(),
			Code:    code,
			Name:    name,
			Enabled: true,
		}
		filter := bson.M{"code": code}
		update := bson.M{"$setOnInsert": set}
		opts := options.Update().SetUpsert(true)
		db.Col("sets").UpdateOne(ctx, filter, update, opts)
	}

	db.Col("user_cards").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "card_id", Value: 1}}, Options: options.Index().SetUnique(true)},
	})
}
