package mispbot

import (
	"fmt"
	"strings"
	"sync"

	"phishing-monitor/pkg/misp"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

// TelegramBot представляет Telegram бота для MISP уведомлений
type TelegramBot struct {
	api        *tgbotapi.BotAPI
	chatIDs    []int64
	mispClient *misp.Client
	logger     *logrus.Logger
	mu         sync.RWMutex
	adminIDs   []int64
}

// NewTelegramBot создает нового Telegram бота
func NewTelegramBot(token string, chatIDs []int64, mispClient *misp.Client) (*TelegramBot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания Telegram бота: %w", err)
	}

	logger := logrus.New()
	logger.Infof("Авторизован как бот: %s", bot.Self.UserName)

	return &TelegramBot{
		api:        bot,
		chatIDs:    chatIDs,
		mispClient: mispClient,
		logger:     logger,
		adminIDs:   chatIDs, // Администраторы = владельцы чатов по умолчанию
	}, nil
}

// SendMessage отправляет сообщение во все настроенные чаты
func (t *TelegramBot) SendMessage(text string) error {
	t.mu.RLock()
	chatIDs := t.chatIDs
	t.mu.RUnlock()

	var lastErr error
	for _, chatID := range chatIDs {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.DisableWebPagePreview = true

		if _, err := t.api.Send(msg); err != nil {
			t.logger.Errorf("Ошибка отправки в чат %d: %v", chatID, err)
			lastErr = err
		}
	}

	return lastErr
}

// SendMessageToChat отправляет сообщение в конкретный чат
func (t *TelegramBot) SendMessageToChat(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true

	_, err := t.api.Send(msg)
	return err
}

// StartCommandHandler запускает обработчик команд бота
func (t *TelegramBot) StartCommandHandler() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := t.api.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			t.handleCommand(update.Message)
		}
	}
}

// handleCommand обрабатывает команды бота
func (t *TelegramBot) handleCommand(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	command := message.Command()
	args := message.CommandArguments()

	t.logger.Infof("Получена команда: /%s от чата %d", command, chatID)

	switch command {
	case "start":
		t.handleStart(chatID)
	case "help":
		t.handleHelp(chatID)
	case "status":
		t.handleStatus(chatID)
	case "events":
		t.handleEvents(chatID, args)
	case "event":
		t.handleEvent(chatID, args)
	case "search":
		t.handleSearch(chatID, args)
	case "subscribe":
		t.handleSubscribe(chatID)
	case "unsubscribe":
		t.handleUnsubscribe(chatID)
	default:
		t.SendMessageToChat(chatID, "Неизвестная команда. Используйте /help для списка команд.")
	}
}

// handleStart обрабатывает команду /start
func (t *TelegramBot) handleStart(chatID int64) {
	message := `
*🛡 MISP SOC Telegram Bot*

Добро пожаловать! Этот бот отправляет уведомления о событиях из платформы MISP.

*Доступные функции:*
• Уведомления о новых событиях
• Уведомления об обновлениях событий
• Уведомления о новых комментариях
• Просмотр последних событий
• Поиск по событиям

Используйте /help для списка команд.
Используйте /subscribe для подписки на уведомления.
`
	t.SendMessageToChat(chatID, message)
}

// handleHelp обрабатывает команду /help
func (t *TelegramBot) handleHelp(chatID int64) {
	message := `
*📋 Список команд:*

/start - Начало работы с ботом
/help - Показать это сообщение
/status - Статус подключения к MISP
/events [N] - Показать последние N событий (по умолчанию 5)
/event [ID] - Показать детали события по ID
/search [запрос] - Поиск событий по названию
/subscribe - Подписаться на уведомления
/unsubscribe - Отписаться от уведомлений

*Примеры:*
• /events 10 - показать 10 последних событий
• /event 241879 - показать событие с ID 241879
• /search phishing - найти события с "phishing"
`
	t.SendMessageToChat(chatID, message)
}

// handleStatus обрабатывает команду /status
func (t *TelegramBot) handleStatus(chatID int64) {
	// Проверяем подключение к MISP
	events, err := t.mispClient.GetRecentEvents(60)
	if err != nil {
		t.SendMessageToChat(chatID, fmt.Sprintf("*❌ Статус MISP:* Ошибка подключения\n```\n%s\n```", err.Error()))
		return
	}

	t.mu.RLock()
	subscribersCount := len(t.chatIDs)
	t.mu.RUnlock()

	message := fmt.Sprintf(`
*✅ Статус системы:*

*MISP API:* Подключено
*Событий за последний час:* %d
*Подписчиков:* %d
*Бот:* @%s
`, len(events), subscribersCount, t.api.Self.UserName)

	t.SendMessageToChat(chatID, message)
}

// handleEvents обрабатывает команду /events
func (t *TelegramBot) handleEvents(chatID int64, args string) {
	limit := 5
	if args != "" {
		fmt.Sscanf(args, "%d", &limit)
		if limit > 20 {
			limit = 20
		}
		if limit < 1 {
			limit = 1
		}
	}

	events, err := t.mispClient.GetEvents()
	if err != nil {
		t.SendMessageToChat(chatID, fmt.Sprintf("Ошибка получения событий: %s", err.Error()))
		return
	}

	if len(events) == 0 {
		t.SendMessageToChat(chatID, "События не найдены.")
		return
	}

	if len(events) > limit {
		events = events[:limit]
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*📊 Последние %d событий:*\n\n", len(events)))

	for i, event := range events {
		orgName := event.Orgc.Name
		if orgName == "" {
			orgName = "N/A"
		}

		sb.WriteString(fmt.Sprintf(
			"*%d.* ID: %s\n    📌 %s\n    🏢 %s | IOC: %s\n\n",
			i+1,
			event.ID,
			escapeMarkdown(truncateString(event.Info, 50)),
			escapeMarkdown(orgName),
			event.AttributeCount,
		))
	}

	t.SendMessageToChat(chatID, sb.String())
}

// handleEvent обрабатывает команду /event
func (t *TelegramBot) handleEvent(chatID int64, args string) {
	if args == "" {
		t.SendMessageToChat(chatID, "Укажите ID события. Пример: /event 241879")
		return
	}

	eventID := strings.TrimSpace(args)
	event, err := t.mispClient.GetEvent(eventID)
	if err != nil {
		t.SendMessageToChat(chatID, fmt.Sprintf("Ошибка получения события: %s", err.Error()))
		return
	}

	// Собираем теги
	var tags []string
	for _, tag := range event.Tags {
		tags = append(tags, tag.Name)
	}
	tagsStr := "Нет тегов"
	if len(tags) > 0 {
		tagsStr = strings.Join(tags[:min(len(tags), 10)], ", ")
		if len(tags) > 10 {
			tagsStr += fmt.Sprintf(" (+%d)", len(tags)-10)
		}
	}

	// Собираем типы атрибутов
	attrTypes := make(map[string]int)
	for _, attr := range event.Attributes {
		attrTypes[attr.Type]++
	}

	var attrTypesStr strings.Builder
	for attrType, count := range attrTypes {
		attrTypesStr.WriteString(fmt.Sprintf("• %s: %d\n", attrType, count))
	}

	orgName := event.Orgc.Name
	if orgName == "" {
		orgName = "Неизвестно"
	}

	message := fmt.Sprintf(`
*📋 Событие #%s*

*Название:* %s
*Создатель:* %s
*Дата события:* %s
*UUID:* %s

*Уровень угрозы:* %s
*Анализ:* %s
*Распределение:* %s
*Опубликовано:* %v

*Теги:*
%s

*Атрибуты (%s):*
%s
`,
		event.ID,
		escapeMarkdown(event.Info),
		escapeMarkdown(orgName),
		event.Date,
		event.UUID,
		misp.ThreatLevelName(event.ThreatLevelID),
		misp.AnalysisLevelName(event.Analysis),
		misp.DistributionName(event.Distribution),
		event.Published,
		escapeMarkdown(tagsStr),
		event.AttributeCount,
		attrTypesStr.String(),
	)

	t.SendMessageToChat(chatID, message)
}

// handleSearch обрабатывает команду /search
func (t *TelegramBot) handleSearch(chatID int64, args string) {
	if args == "" {
		t.SendMessageToChat(chatID, "Укажите поисковый запрос. Пример: /search phishing")
		return
	}

	query := strings.ToLower(strings.TrimSpace(args))

	events, err := t.mispClient.GetEvents()
	if err != nil {
		t.SendMessageToChat(chatID, fmt.Sprintf("Ошибка поиска: %s", err.Error()))
		return
	}

	// Фильтруем события по запросу
	var filtered []misp.Event
	for _, event := range events {
		if strings.Contains(strings.ToLower(event.Info), query) {
			filtered = append(filtered, event)
		}
	}

	if len(filtered) == 0 {
		t.SendMessageToChat(chatID, fmt.Sprintf("События по запросу \"%s\" не найдены.", args))
		return
	}

	if len(filtered) > 10 {
		filtered = filtered[:10]
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*🔍 Результаты поиска \"%s\":*\n\n", escapeMarkdown(args)))

	for i, event := range filtered {
		orgName := event.Orgc.Name
		if orgName == "" {
			orgName = "N/A"
		}

		sb.WriteString(fmt.Sprintf(
			"*%d.* ID: %s\n    📌 %s\n    🏢 %s\n\n",
			i+1,
			event.ID,
			escapeMarkdown(truncateString(event.Info, 50)),
			escapeMarkdown(orgName),
		))
	}

	t.SendMessageToChat(chatID, sb.String())
}

// handleSubscribe обрабатывает команду /subscribe
func (t *TelegramBot) handleSubscribe(chatID int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Проверяем, не подписан ли уже
	for _, id := range t.chatIDs {
		if id == chatID {
			t.SendMessageToChat(chatID, "✅ Вы уже подписаны на уведомления.")
			return
		}
	}

	t.chatIDs = append(t.chatIDs, chatID)
	t.SendMessageToChat(chatID, "✅ Вы успешно подписались на уведомления о событиях MISP.")
}

// handleUnsubscribe обрабатывает команду /unsubscribe
func (t *TelegramBot) handleUnsubscribe(chatID int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i, id := range t.chatIDs {
		if id == chatID {
			t.chatIDs = append(t.chatIDs[:i], t.chatIDs[i+1:]...)
			t.SendMessageToChat(chatID, "❌ Вы отписались от уведомлений.")
			return
		}
	}

	t.SendMessageToChat(chatID, "Вы не были подписаны на уведомления.")
}

// truncateString обрезает строку до указанной длины
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// min возвращает минимальное из двух чисел
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
