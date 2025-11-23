package main

import (
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"phishing-monitor/pkg/misp"
	"phishing-monitor/pkg/mispbot"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	logger.Info("🚀 Запуск MISP SOC Telegram Bot...")

	// Загружаем переменные окружения
	if err := godotenv.Load(); err != nil {
		logger.Warn("Файл .env не найден, используем системные переменные окружения")
	}

	// Получаем конфигурацию
	config, err := loadConfig()
	if err != nil {
		logger.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	logger.Info("✅ Конфигурация загружена")

	// Создаем MISP клиент
	mispClient := misp.NewClient(
		config.MISPBaseURL,
		config.MISPAPIKey,
		config.MISPSkipTLSVerify,
	)

	logger.Info("✅ MISP клиент создан")

	// Создаем Telegram бота
	telegramBot, err := mispbot.NewTelegramBot(
		config.TelegramToken,
		config.TelegramChatIDs,
		mispClient,
	)
	if err != nil {
		logger.Fatalf("Ошибка создания Telegram бота: %v", err)
	}

	logger.Info("✅ Telegram бот создан")

	// Создаем монитор
	monitor := mispbot.NewMonitor(
		mispClient,
		telegramBot,
		config.PollIntervalSeconds,
	)

	// Запускаем обработчик команд бота в отдельной горутине
	go telegramBot.StartCommandHandler()
	logger.Info("✅ Обработчик команд запущен")

	// Запускаем монитор в отдельной горутине
	go monitor.Start()
	logger.Info("✅ Мониторинг MISP событий запущен")

	// Отправляем уведомление о запуске
	startupMessage := `
*🟢 MISP SOC Bot запущен*

Бот начал мониторинг событий MISP.
Используйте /help для списка команд.
`
	telegramBot.SendMessage(startupMessage)

	// Ожидаем сигнал завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	logger.Info("Получен сигнал завершения, останавливаем бота...")

	// Останавливаем монитор
	monitor.Stop()

	// Отправляем уведомление об остановке
	telegramBot.SendMessage("*🔴 MISP SOC Bot остановлен*")

	logger.Info("👋 Бот остановлен")
}

// Config содержит конфигурацию приложения
type Config struct {
	// MISP настройки
	MISPBaseURL        string
	MISPAPIKey         string
	MISPSkipTLSVerify  bool

	// Telegram настройки
	TelegramToken   string
	TelegramChatIDs []int64

	// Настройки мониторинга
	PollIntervalSeconds int
}

// loadConfig загружает конфигурацию из переменных окружения
func loadConfig() (*Config, error) {
	config := &Config{
		MISPBaseURL:         getEnv("MISP_BASE_URL", "https://misp.local"),
		MISPAPIKey:          getEnv("MISP_API_KEY", ""),
		MISPSkipTLSVerify:   getEnvBool("MISP_SKIP_TLS_VERIFY", true),
		TelegramToken:       getEnv("TELEGRAM_BOT_TOKEN", ""),
		PollIntervalSeconds: getEnvInt("POLL_INTERVAL_SECONDS", 60),
	}

	// Парсим Chat IDs
	chatIDsStr := getEnv("TELEGRAM_CHAT_IDS", "")
	if chatIDsStr != "" {
		ids := strings.Split(chatIDsStr, ",")
		for _, idStr := range ids {
			idStr = strings.TrimSpace(idStr)
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				config.TelegramChatIDs = append(config.TelegramChatIDs, id)
			}
		}
	}

	// Валидация
	if config.MISPAPIKey == "" {
		logrus.Fatal("MISP_API_KEY не установлен")
	}
	if config.TelegramToken == "" {
		logrus.Fatal("TELEGRAM_BOT_TOKEN не установлен")
	}

	return config, nil
}

// getEnv получает переменную окружения или возвращает значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool получает булеву переменную окружения
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}

// getEnvInt получает целочисленную переменную окружения
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
