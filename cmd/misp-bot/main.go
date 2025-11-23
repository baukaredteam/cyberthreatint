package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"phishing-monitor/pkg/misp"
	"phishing-monitor/pkg/mispbot"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	// Настраиваем логгер
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     true,
	})

	// Красивый баннер при запуске
	printBanner()

	logger.Info("========================================")
	logger.Info("   MISP SOC Telegram Bot v1.0")
	logger.Info("   Запуск приложения...")
	logger.Info("========================================")

	// Загружаем переменные окружения
	if err := godotenv.Load(); err != nil {
		logger.Warn("[CONFIG] Файл .env не найден, используем системные переменные окружения")
	} else {
		logger.Info("[CONFIG] Файл .env загружен успешно")
	}

	// Получаем конфигурацию
	config, err := loadConfig(logger)
	if err != nil {
		logger.Fatalf("[ERROR] Ошибка загрузки конфигурации: %v", err)
	}

	logger.Info("[CONFIG] Конфигурация загружена:")
	logger.Infof("[CONFIG]   MISP URL: %s", config.MISPBaseURL)
	logger.Infof("[CONFIG]   Poll Interval: %d секунд", config.PollIntervalSeconds)
	logger.Infof("[CONFIG]   Chat IDs: %v", config.TelegramChatIDs)

	// Создаем MISP клиент
	logger.Info("[INIT] Создание MISP клиента...")
	mispClient := misp.NewClient(
		config.MISPBaseURL,
		config.MISPAPIKey,
		config.MISPSkipTLSVerify,
	)
	logger.Info("[INIT] MISP клиент создан успешно")

	// Проверяем подключение к MISP
	logger.Info("[INIT] Проверка подключения к MISP API...")
	if _, err := mispClient.GetEvents(); err != nil {
		logger.Errorf("[ERROR] Не удалось подключиться к MISP: %v", err)
		logger.Warn("[INIT] Продолжаем запуск, подключение будет повторено...")
	} else {
		logger.Info("[INIT] Подключение к MISP API успешно!")
	}

	// Создаем Telegram бота
	logger.Info("[INIT] Создание Telegram бота...")
	telegramBot, err := mispbot.NewTelegramBot(
		config.TelegramToken,
		config.TelegramChatIDs,
		mispClient,
		logger,
	)
	if err != nil {
		logger.Fatalf("[ERROR] Ошибка создания Telegram бота: %v", err)
	}
	logger.Info("[INIT] Telegram бот создан успешно")

	// Создаем монитор
	logger.Info("[INIT] Создание монитора MISP событий...")
	monitor := mispbot.NewMonitor(
		mispClient,
		telegramBot,
		config.PollIntervalSeconds,
		logger,
	)
	logger.Info("[INIT] Монитор создан успешно")

	// Запускаем обработчик команд бота в отдельной горутине
	go telegramBot.StartCommandHandler()

	// Отправляем уведомление о запуске
	startupMessage := fmt.Sprintf(`
*🟢 MISP SOC Bot запущен*

Бот начал мониторинг событий MISP.
Время запуска: %s
Используйте /help для списка команд.
`, time.Now().Format("02.01.2006 15:04:05"))
	telegramBot.SendMessage(startupMessage)

	logger.Info("========================================")
	logger.Info("   Все компоненты инициализированы!")
	logger.Info("   Бот работает в режиме мониторинга")
	logger.Info("========================================")
	logger.Info("")

	// Запускаем монитор (бесконечный цикл)
	// Эта функция никогда не вернется
	monitor.Start()
}

// printBanner выводит ASCII баннер
func printBanner() {
	banner := `
╔══════════════════════════════════════════════════════════════╗
║                                                              ║
║   ███╗   ███╗██╗███████╗██████╗     ██████╗  ██████╗ ████████╗║
║   ████╗ ████║██║██╔════╝██╔══██╗    ██╔══██╗██╔═══██╗╚══██╔══╝║
║   ██╔████╔██║██║███████╗██████╔╝    ██████╔╝██║   ██║   ██║   ║
║   ██║╚██╔╝██║██║╚════██║██╔═══╝     ██╔══██╗██║   ██║   ██║   ║
║   ██║ ╚═╝ ██║██║███████║██║         ██████╔╝╚██████╔╝   ██║   ║
║   ╚═╝     ╚═╝╚═╝╚══════╝╚═╝         ╚═════╝  ╚═════╝    ╚═╝   ║
║                                                              ║
║            SOC Team - Threat Intelligence Monitor            ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
}

// Config содержит конфигурацию приложения
type Config struct {
	// MISP настройки
	MISPBaseURL       string
	MISPAPIKey        string
	MISPSkipTLSVerify bool

	// Telegram настройки
	TelegramToken   string
	TelegramChatIDs []int64

	// Настройки мониторинга
	PollIntervalSeconds int
}

// loadConfig загружает конфигурацию из переменных окружения
func loadConfig(logger *logrus.Logger) (*Config, error) {
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
		logger.Error("[CONFIG] MISP_API_KEY не установлен!")
		return nil, fmt.Errorf("MISP_API_KEY не установлен")
	}
	if config.TelegramToken == "" {
		logger.Error("[CONFIG] TELEGRAM_BOT_TOKEN не установлен!")
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN не установлен")
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
