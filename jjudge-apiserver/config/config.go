package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort int
	Database   DatabaseConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

func LoadConfig() Config {
	if os.Getenv("ENV") == "dev" {
		godotenv.Load()
	}

	dbConfig := DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnvInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", "jjudge"),
		Password: getEnv("DB_PASSWORD", "password"),
		DBName:   getEnv("DB_NAME", "jjudge_db"),
	}

	return Config{
		ServerPort: getEnvInt("SERVER_PORT", 8080),
		Database:   dbConfig,
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if valueStr, exists := os.LookupEnv(key); exists {
		var value int
		fmt.Sscanf(valueStr, "%d", &value)
		return value
	}
	return defaultValue
}
