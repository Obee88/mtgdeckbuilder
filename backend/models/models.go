package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ─── User ────────────────────────────────────────────────────────────────────

type User struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username    string             `bson:"username" json:"username"`
	Password    string             `bson:"password" json:"-"`
	IsAdmin     bool               `bson:"is_admin" json:"is_admin"`
	JAD         int                `bson:"jad" json:"jad"`
	JADLocked   int                `bson:"jad_locked" json:"jad_locked"` // sum of active bids
	RegisteredAt time.Time         `bson:"registered_at" json:"registered_at"`
	BoostersOpened int             `bson:"boosters_opened" json:"boosters_opened"`
}

// ─── Card (master Scryfall data) ─────────────────────────────────────────────

type Card struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ScryfallID    string             `bson:"scryfall_id" json:"scryfall_id"`
	Name          string             `bson:"name" json:"name"`
	SetCode       string             `bson:"set_code" json:"set_code"`
	SetName       string             `bson:"set_name" json:"set_name"`
	Rarity        string             `bson:"rarity" json:"rarity"` // common, uncommon, rare, mythic
	ImageURI      string             `bson:"image_uri" json:"image_uri"`
	ManaCost      string             `bson:"mana_cost" json:"mana_cost"`
	TypeLine      string             `bson:"type_line" json:"type_line"`
	OracleText    string             `bson:"oracle_text" json:"oracle_text"`
	ColorIdentity []string           `bson:"color_identity" json:"color_identity"`
	SetEnabled    bool               `bson:"set_enabled" json:"set_enabled"`
	Banned        bool               `bson:"banned" json:"banned"`
}

// ─── UserCard (owned cards) ───────────────────────────────────────────────────

type UserCard struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID   primitive.ObjectID `bson:"user_id" json:"user_id"`
	CardID   primitive.ObjectID `bson:"card_id" json:"card_id"`
	CardName string             `bson:"card_name" json:"card_name"`
	SetCode  string             `bson:"set_code" json:"set_code"`
	SetName  string             `bson:"set_name" json:"set_name"`
	Rarity   string             `bson:"rarity" json:"rarity"`
	ImageURI string             `bson:"image_uri" json:"image_uri"`
	ManaCost string             `bson:"mana_cost" json:"mana_cost"`
	TypeLine string             `bson:"type_line" json:"type_line"`
	Quantity int                `bson:"quantity" json:"quantity"`
}

// ─── Deck ─────────────────────────────────────────────────────────────────────

type DeckCard struct {
	CardID   primitive.ObjectID `bson:"card_id" json:"card_id"`
	CardName string             `bson:"card_name" json:"card_name"`
	ImageURI string             `bson:"image_uri" json:"image_uri"`
	Quantity int                `bson:"quantity" json:"quantity"`
}

type Deck struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	Name      string             `bson:"name" json:"name"`
	Cards     []DeckCard         `bson:"cards" json:"cards"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// ─── Booster History ──────────────────────────────────────────────────────────

type BoosterHistory struct {
	ID       primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	UserID   primitive.ObjectID   `bson:"user_id" json:"user_id"`
	OpenedAt time.Time            `bson:"opened_at" json:"opened_at"`
	Cards    []primitive.ObjectID `bson:"cards" json:"cards"`
}

// ─── MTG Set ──────────────────────────────────────────────────────────────────

type MTGSet struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Code    string             `bson:"code" json:"code"`
	Name    string             `bson:"set_name" json:"set_name"`
	Enabled bool               `bson:"enabled" json:"enabled"`
}

// ─── Market ───────────────────────────────────────────────────────────────────

type MarketCard struct {
	ID              primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	CardID          primitive.ObjectID  `bson:"card_id" json:"card_id"`
	CardName        string              `bson:"card_name" json:"card_name"`
	ImageURI        string              `bson:"image_uri" json:"image_uri"`
	Rarity          string              `bson:"rarity" json:"rarity"`
	ManaCost        string              `bson:"mana_cost" json:"mana_cost"`
	TypeLine        string              `bson:"type_line" json:"type_line"`
	CurrentBid      int                 `bson:"current_bid" json:"current_bid"`
	CurrentBidderID *primitive.ObjectID `bson:"current_bidder_id,omitempty" json:"current_bidder_id,omitempty"`
	CurrentBidder   string              `bson:"current_bidder,omitempty" json:"current_bidder,omitempty"`
	BidExpiresAt    *time.Time          `bson:"bid_expires_at,omitempty" json:"bid_expires_at,omitempty"`
	HaterIDs        []primitive.ObjectID `bson:"hater_ids" json:"hater_ids"`
	HateCount       int                 `bson:"hate_count" json:"hate_count"`
	Status          string              `bson:"status" json:"status"` // active, won
	AddedAt         time.Time           `bson:"added_at" json:"added_at"`
}

// ─── Trade ────────────────────────────────────────────────────────────────────

type Trade struct {
	ID             primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	FromUserID     primitive.ObjectID   `bson:"from_user_id" json:"from_user_id"`
	FromUsername   string               `bson:"from_username" json:"from_username"`
	ToUserID       primitive.ObjectID   `bson:"to_user_id" json:"to_user_id"`
	ToUsername     string               `bson:"to_username" json:"to_username"`
	OfferedCards   []TradeCard          `bson:"offered_cards" json:"offered_cards"`
	OfferedJAD     int                  `bson:"offered_jad" json:"offered_jad"`
	RequestedCards []TradeCard          `bson:"requested_cards" json:"requested_cards"`
	RequestedJAD   int                  `bson:"requested_jad" json:"requested_jad"`
	Status         string               `bson:"status" json:"status"` // pending, accepted, declined, cancelled
	CreatedAt      time.Time            `bson:"created_at" json:"created_at"`
}

type TradeCard struct {
	UserCardID primitive.ObjectID `bson:"user_card_id" json:"user_card_id"`
	CardName   string             `bson:"card_name" json:"card_name"`
	ImageURI   string             `bson:"image_uri" json:"image_uri"`
	Quantity   int                `bson:"quantity" json:"quantity"`
}

// ─── Card List ────────────────────────────────────────────────────────────────

type CardList struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	Name      string             `bson:"name" json:"name"`
	CardNames []string           `bson:"card_names" json:"card_names"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// ─── Ban ──────────────────────────────────────────────────────────────────────

type BannedCard struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CardName string             `bson:"card_name" json:"card_name"`
	Reason   string             `bson:"reason" json:"reason"` // hates, manual
	BannedAt time.Time          `bson:"banned_at" json:"banned_at"`
}
