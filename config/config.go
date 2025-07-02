package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Config struct {
	TelegramBotToken string
	TelegramChatID   int64
	CertstreamURL    string
	DatabasePath     string
	LogLevel         string
	ClientDomains    []string
	UpdateInterval   int // minutes
	SimilarityThreshold float64
}

func Load() *Config {
	// Загружаем .env файл если он существует
	if err := godotenv.Load(); err != nil {
		logrus.Warn("Файл .env не найден, используем переменные окружения")
	}

	chatID, _ := strconv.ParseInt(os.Getenv("TELEGRAM_CHAT_ID"), 10, 64)
	updateInterval, _ := strconv.Atoi(getEnvOrDefault("UPDATE_INTERVAL", "60"))
	similarityThreshold, _ := strconv.ParseFloat(getEnvOrDefault("SIMILARITY_THRESHOLD", "0.8"), 64)

	clientDomainsStr := getEnvOrDefault("CLIENT_DOMAINS", "qazpost.kz,example.com")
	clientDomains := strings.Split(clientDomainsStr, ",")

	return &Config{
		TelegramBotToken:    os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID:      chatID,
		CertstreamURL:       getEnvOrDefault("CERTSTREAM_URL", "wss://certstream.calidog.io"),
		DatabasePath:        getEnvOrDefault("DATABASE_PATH", "./data/phishing.db"),
		LogLevel:            getEnvOrDefault("LOG_LEVEL", "info"),
		ClientDomains:       clientDomains,
		UpdateInterval:      updateInterval,
		SimilarityThreshold: similarityThreshold,
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}