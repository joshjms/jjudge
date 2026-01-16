package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort int
	Database   DatabaseConfig
	Minio      MinioConfig
	GCS        GCSConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	UseSSL   bool
}

type MinioConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type GCSConfig struct {
	Bucket          string
	ProjectID       string
	CredentialsFile string
}

func LoadConfig() Config {
	if os.Getenv("ENV") == "dev" {
		godotenv.Load()
	}

	return Config{
		ServerPort: getEnvInt("SERVER_PORT", 8080),
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "jjudge"),
			Password: getEnv("DB_PASSWORD", "jjudge"),
			DBName:   getEnv("DB_NAME", "jjudge"),
			UseSSL:   getEnv("DB_USE_SSL", "false") == "true",
		},
		Minio: MinioConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY", ""),
			SecretKey: getEnv("MINIO_SECRET_KEY", ""),
			Bucket:    getEnv("MINIO_BUCKET", "jjudge"),
			UseSSL:    getEnv("MINIO_USE_SSL", "false") == "true",
		},
		GCS: GCSConfig{
			Bucket:          getEnv("GCS_BUCKET", ""),
			ProjectID:       getEnv("GCS_PROJECT_ID", ""),
			CredentialsFile: getEnv("GCS_CREDENTIALS_FILE", ""),
		},
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
