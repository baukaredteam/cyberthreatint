package mispbot

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"phishing-monitor/pkg/misp"

	"github.com/sirupsen/logrus"
)

// Monitor отслеживает изменения в MISP и отправляет уведомления
type Monitor struct {
	client           *misp.Client
	bot              *TelegramBot
	pollInterval     time.Duration
	lastCheck        int64
	knownEvents      map[string]*EventState
	mu               sync.RWMutex
	logger           *logrus.Logger
	stopChan         chan struct{}
	running          bool
}

// EventState хранит состояние события для отслеживания изменений
type EventState struct {
	EventID        string
	LastTimestamp  string
	AttributeCount int
	LastModified   time.Time
	CommentCount   int
}

// NewMonitor создает новый монитор MISP
func NewMonitor(client *misp.Client, bot *TelegramBot, pollIntervalSeconds int) *Monitor {
	return &Monitor{
		client:       client,
		bot:          bot,
		pollInterval: time.Duration(pollIntervalSeconds) * time.Second,
		knownEvents:  make(map[string]*EventState),
		logger:       logrus.New(),
		stopChan:     make(chan struct{}),
	}
}

// Start запускает мониторинг
func (m *Monitor) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.mu.Unlock()

	m.logger.Info("Запуск мониторинга MISP событий...")

	// Первоначальная загрузка событий
	if err := m.loadInitialEvents(); err != nil {
		m.logger.Errorf("Ошибка загрузки начальных событий: %v", err)
	}

	m.lastCheck = time.Now().Unix()

	// Основной цикл мониторинга
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkForUpdates()
		case <-m.stopChan:
			m.logger.Info("Остановка мониторинга MISP")
			return
		}
	}
}

// Stop останавливает мониторинг
func (m *Monitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		close(m.stopChan)
		m.running = false
	}
}

// loadInitialEvents загружает текущие события при старте
func (m *Monitor) loadInitialEvents() error {
	events, err := m.client.GetEvents()
	if err != nil {
		return fmt.Errorf("ошибка получения событий: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, event := range events {
		attrCount, _ := strconv.Atoi(event.AttributeCount)
		m.knownEvents[event.ID] = &EventState{
			EventID:        event.ID,
			LastTimestamp:  event.Timestamp,
			AttributeCount: attrCount,
			LastModified:   time.Now(),
		}
	}

	m.logger.Infof("Загружено %d событий для мониторинга", len(events))
	return nil
}

// checkForUpdates проверяет наличие обновлений
func (m *Monitor) checkForUpdates() {
	m.logger.Debug("Проверка обновлений MISP...")

	events, err := m.client.GetEvents()
	if err != nil {
		m.logger.Errorf("Ошибка получения событий: %v", err)
		return
	}

	for _, event := range events {
		m.processEvent(event)
	}

	m.lastCheck = time.Now().Unix()
}

// processEvent обрабатывает событие и определяет тип обновления
func (m *Monitor) processEvent(event misp.Event) {
	m.mu.Lock()
	existingState, exists := m.knownEvents[event.ID]
	m.mu.Unlock()

	if !exists {
		// Новое событие
		m.handleNewEvent(event)
		return
	}

	// Проверяем изменения в существующем событии
	if event.Timestamp != existingState.LastTimestamp {
		m.handleEventUpdate(event, existingState)
	}
}

// handleNewEvent обрабатывает новое событие
func (m *Monitor) handleNewEvent(event misp.Event) {
	m.logger.Infof("Обнаружено новое событие: ID=%s, Info=%s", event.ID, event.Info)

	// Получаем детальную информацию о событии
	eventDetail, err := m.client.GetEvent(event.ID)
	if err != nil {
		m.logger.Errorf("Ошибка получения деталей события %s: %v", event.ID, err)
		return
	}

	// Формируем и отправляем уведомление
	message := m.formatNewEventMessage(eventDetail)
	if err := m.bot.SendMessage(message); err != nil {
		m.logger.Errorf("Ошибка отправки уведомления: %v", err)
	}

	// Сохраняем состояние события
	attrCount, _ := strconv.Atoi(event.AttributeCount)
	m.mu.Lock()
	m.knownEvents[event.ID] = &EventState{
		EventID:        event.ID,
		LastTimestamp:  event.Timestamp,
		AttributeCount: attrCount,
		LastModified:   time.Now(),
	}
	m.mu.Unlock()
}

// handleEventUpdate обрабатывает обновление события
func (m *Monitor) handleEventUpdate(event misp.Event, oldState *EventState) {
	m.logger.Infof("Обнаружено обновление события: ID=%s", event.ID)

	// Получаем детальную информацию
	eventDetail, err := m.client.GetEvent(event.ID)
	if err != nil {
		m.logger.Errorf("Ошибка получения деталей события %s: %v", event.ID, err)
		return
	}

	// Определяем тип изменения
	newAttrCount, _ := strconv.Atoi(event.AttributeCount)

	var message string
	if newAttrCount > oldState.AttributeCount {
		// Добавлены новые атрибуты (возможно комментарии)
		message = m.formatEventUpdateMessage(eventDetail, oldState, newAttrCount-oldState.AttributeCount)
	} else {
		// Общее обновление события
		message = m.formatEventModifiedMessage(eventDetail)
	}

	if err := m.bot.SendMessage(message); err != nil {
		m.logger.Errorf("Ошибка отправки уведомления: %v", err)
	}

	// Обновляем состояние
	m.mu.Lock()
	m.knownEvents[event.ID] = &EventState{
		EventID:        event.ID,
		LastTimestamp:  event.Timestamp,
		AttributeCount: newAttrCount,
		LastModified:   time.Now(),
	}
	m.mu.Unlock()
}

// formatNewEventMessage форматирует сообщение о новом событии
func (m *Monitor) formatNewEventMessage(event *misp.EventDetail) string {
	// Собираем теги
	var tags []string
	for _, tag := range event.Tags {
		tags = append(tags, tag.Name)
	}
	tagsStr := "Нет тегов"
	if len(tags) > 0 {
		tagsStr = ""
		for i, tag := range tags {
			if i > 0 {
				tagsStr += ", "
			}
			tagsStr += tag
			if i >= 4 {
				tagsStr += fmt.Sprintf(" (+%d)", len(tags)-5)
				break
			}
		}
	}

	// Форматируем timestamp
	timestamp, _ := strconv.ParseInt(event.Timestamp, 10, 64)
	modifiedTime := time.Unix(timestamp, 0).Format("02.01.2006 15:04:05")

	publishTime := "Не опубликовано"
	if event.Published {
		pubTimestamp, _ := strconv.ParseInt(event.PublishTimestamp, 10, 64)
		publishTime = time.Unix(pubTimestamp, 0).Format("02.01.2006 15:04:05")
	}

	// Определяем создателя (организацию)
	creatorOrg := event.Orgc.Name
	if creatorOrg == "" {
		creatorOrg = "Неизвестно"
	}

	message := fmt.Sprintf(`
*🆕 НОВОЕ СОБЫТИЕ MISP*

*ID события:* %s
*Кем создан:* %s
*Название события:* %s
*Теги:* %s
*IOC:* %s
*Уровень угрозы:* %s
*Анализ:* %s
*Распределение:* %s
*Дата события:* %s
*Дата публикации:* %s
*Дата редактирования:* %s
`,
		event.ID,
		escapeMarkdown(creatorOrg),
		escapeMarkdown(event.Info),
		escapeMarkdown(tagsStr),
		event.AttributeCount,
		misp.ThreatLevelName(event.ThreatLevelID),
		misp.AnalysisLevelName(event.Analysis),
		misp.DistributionName(event.Distribution),
		event.Date,
		publishTime,
		modifiedTime,
	)

	return message
}

// formatEventUpdateMessage форматирует сообщение об обновлении события
func (m *Monitor) formatEventUpdateMessage(event *misp.EventDetail, oldState *EventState, newAttrsCount int) string {
	timestamp, _ := strconv.ParseInt(event.Timestamp, 10, 64)
	modifiedTime := time.Unix(timestamp, 0).Format("02.01.2006 15:04:05")

	// Ищем новые комментарии
	var recentComments []string
	for _, attr := range event.Attributes {
		if attr.Type == "comment" || attr.Category == "Internal reference" {
			attrTimestamp, _ := strconv.ParseInt(attr.Timestamp, 10, 64)
			if attrTimestamp > oldState.LastModified.Unix()-60 {
				recentComments = append(recentComments, attr.Value)
			}
		}
	}

	commentsInfo := ""
	if len(recentComments) > 0 {
		commentsInfo = "\n*Новые комментарии:*\n"
		for _, comment := range recentComments {
			if len(comment) > 200 {
				comment = comment[:200] + "..."
			}
			commentsInfo += fmt.Sprintf("• %s\n", escapeMarkdown(comment))
		}
	}

	creatorOrg := event.Orgc.Name
	if creatorOrg == "" {
		creatorOrg = "Неизвестно"
	}

	message := fmt.Sprintf(`
*📝 ОБНОВЛЕНИЕ СОБЫТИЯ MISP*

*ID события:* %s
*Кем создано:* %s
*Название события:* %s
*Когда событие было создано:* %s
*Дата последнего редактирования:* %s
*IOCs:* %s (добавлено: +%d)
%s`,
		event.ID,
		escapeMarkdown(creatorOrg),
		escapeMarkdown(event.Info),
		event.Date,
		modifiedTime,
		event.AttributeCount,
		newAttrsCount,
		commentsInfo,
	)

	return message
}

// formatEventModifiedMessage форматирует сообщение о модификации события
func (m *Monitor) formatEventModifiedMessage(event *misp.EventDetail) string {
	timestamp, _ := strconv.ParseInt(event.Timestamp, 10, 64)
	modifiedTime := time.Unix(timestamp, 0).Format("02.01.2006 15:04:05")

	creatorOrg := event.Orgc.Name
	if creatorOrg == "" {
		creatorOrg = "Неизвестно"
	}

	message := fmt.Sprintf(`
*🔄 ИЗМЕНЕНИЕ СОБЫТИЯ MISP*

*ID события:* %s
*Кем создано:* %s
*Название события:* %s
*Когда событие было создано:* %s
*Дата последнего редактирования:* %s
*IOCs:* %s
*Уровень угрозы:* %s
*Распределение:* %s
`,
		event.ID,
		escapeMarkdown(creatorOrg),
		escapeMarkdown(event.Info),
		event.Date,
		modifiedTime,
		event.AttributeCount,
		misp.ThreatLevelName(event.ThreatLevelID),
		misp.DistributionName(event.Distribution),
	)

	return message
}

// escapeMarkdown экранирует специальные символы Markdown
func escapeMarkdown(s string) string {
	replacer := []string{
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	}

	result := s
	for i := 0; i < len(replacer); i += 2 {
		result = replaceAll(result, replacer[i], replacer[i+1])
	}
	return result
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		if string(s[i]) == old {
			result += new
		} else {
			result += string(s[i])
		}
	}
	return result
}
