package config

import (
	"os"
	"strings"
)

type Config struct {
	MongoURI       string
	JWTSecret      string
	Port           string
	IsProd         bool
	AllowedOrigins []string
}

func Load() *Config {
	isProd := os.Getenv("ENV") == "prod"
	mongoURI := "mongodb://localhost:27017"
	if isProd {
		mongoURI = os.Getenv("DB_URI")
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "mtgdeckbuilder-secret-key-change-in-prod"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}
	allowedOrigins := []string{"http://localhost:5200", "http://localhost:5173", "http://localhost:3000"}
	if raw := os.Getenv("ALLOWED_ORIGINS"); raw != "" {
		allowedOrigins = strings.Split(raw, ",")
	}
	return &Config{
		MongoURI:       mongoURI,
		JWTSecret:      jwtSecret,
		Port:           port,
		IsProd:         isProd,
		AllowedOrigins: allowedOrigins,
	}
}
