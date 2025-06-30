package config

import (
	"os"
)

type Config struct {
	DBConnStr  string
	JWTSecret  []byte
	ServerPort string
}

func LoadConfig() *Config {
	return &Config{
		DBConnStr:  getEnvOrDefault("DB_CONN", "host=localhost port=5432 user=postgres password=2006Hjvfy! dbname=playboxdb sslmode=disable"),
		JWTSecret:  []byte(getEnvOrDefault("JWT_SECRET", "")),
		ServerPort: getEnvOrDefault("PORT", "8080"),
	}
}

func getEnvOrDefault(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}
