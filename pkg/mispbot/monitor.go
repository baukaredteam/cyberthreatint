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
	knownComments    map[string]map[string]bool // eventID -> commentID -> exists
	mu               sync.RWMutex
	logger           *logrus.Logger
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
func NewMonitor(client *misp.Client, bot *TelegramBot, pollIntervalSeconds int, logger *logrus.Logger) *Monitor {
	return &Monitor{
		client:        client,
		bot:           bot,
		pollInterval:  time.Duration(pollIntervalSeconds) * time.Second,
		knownEvents:   make(map[string]*EventState),
		knownComments: make(map[string]map[string]bool),
		logger:        logger,
	}
}

// Start запускает мониторинг (бесконечный цикл)
func (m *Monitor) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.mu.Unlock()

	m.logger.Info("========================================")
	m.logger.Info("   MISP SOC Monitor запущен")
	m.logger.Info("========================================")
	m.logger.Infof("Интервал опроса: %v", m.pollInterval)

	// Первоначальная загрузка событий
	m.logger.Info("[INIT] Загрузка существующих событий из MISP...")
	if err := m.loadInitialEvents(); err != nil {
		m.logger.Errorf("[ERROR] Ошибка загрузки начальных событий: %v", err)
		m.logger.Info("[RETRY] Повторная попытка через 10 секунд...")
		time.Sleep(10 * time.Second)
	}

	m.lastCheck = time.Now().Unix()

	// Бесконечный цикл мониторинга
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	cycleCount := 0
	for {
		select {
		case <-ticker.C:
			cycleCount++
			m.logger.Info("----------------------------------------")
			m.logger.Infof("[CYCLE #%d] Начало цикла проверки | %s", cycleCount, time.Now().Format("15:04:05"))
			m.checkForUpdates()
			m.logger.Infof("[CYCLE #%d] Цикл завершен | Следующая проверка через %v", cycleCount, m.pollInterval)
		}
	}
}

// loadInitialEvents загружает текущие события при старте
func (m *Monitor) loadInitialEvents() error {
	m.logger.Info("[FETCH] Получение списка событий из MISP API...")

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
		m.knownComments[event.ID] = make(map[string]bool)
	}

	m.logger.Infof("[INIT] Загружено %d событий для мониторинга", len(events))
	m.logger.Info("[INIT] Инициализация завершена успешно")
	return nil
}

// checkForUpdates проверяет наличие обновлений
func (m *Monitor) checkForUpdates() {
	// Проверка новых событий
	m.logger.Info("[SEARCH] Идет поиск новых событий...")

	events, err := m.client.GetEvents()
	if err != nil {
		m.logger.Errorf("[ERROR] Ошибка получения событий: %v", err)
		return
	}

	m.logger.Infof("[FETCH] Получено %d событий из MISP", len(events))

	newEventsCount := 0
	updatedEventsCount := 0

	for _, event := range events {
		isNew, isUpdated := m.processEvent(event)
		if isNew {
			newEventsCount++
		}
		if isUpdated {
			updatedEventsCount++
		}
	}

	if newEventsCount > 0 {
		m.logger.Infof("[NEW] Найдено новых событий: %d", newEventsCount)
	} else {
		m.logger.Info("[NEW] Новых событий не найдено")
	}

	// Проверка комментариев
	m.logger.Info("[SEARCH] Идет поиск новых комментариев...")
	m.checkForNewComments()

	if updatedEventsCount > 0 {
		m.logger.Infof("[UPDATE] Обновлено событий: %d", updatedEventsCount)
	} else {
		m.logger.Info("[UPDATE] Изменений в событиях не найдено")
	}

	m.lastCheck = time.Now().Unix()
}

// checkForNewComments проверяет новые комментарии во всех событиях
func (m *Monitor) checkForNewComments() {
	m.mu.RLock()
	eventIDs := make([]string, 0, len(m.knownEvents))
	for id := range m.knownEvents {
		eventIDs = append(eventIDs, id)
	}
	m.mu.RUnlock()

	newCommentsCount := 0

	for _, eventID := range eventIDs {
		comments, err := m.client.GetEventComments(eventID)
		if err != nil {
			continue
		}

		m.mu.Lock()
		if m.knownComments[eventID] == nil {
			m.knownComments[eventID] = make(map[string]bool)
		}

		for _, comment := range comments {
			if !m.knownComments[eventID][comment.ID] {
				// Новый комментарий найден
				m.knownComments[eventID][comment.ID] = true
				newCommentsCount++

				m.logger.Infof("[COMMENT] Новый комментарий в событии #%s", eventID)

				// Отправляем уведомление
				go m.notifyNewComment(eventID, comment)
			}
		}
		m.mu.Unlock()
	}

	if newCommentsCount > 0 {
		m.logger.Infof("[COMMENT] Найдено новых комментариев: %d", newCommentsCount)
	} else {
		m.logger.Info("[COMMENT] Новых комментариев не найдено")
	}
}

// notifyNewComment отправляет уведомление о новом комментарии
func (m *Monitor) notifyNewComment(eventID string, comment misp.Attribute) {
	event, err := m.client.GetEvent(eventID)
	if err != nil {
		m.logger.Errorf("[ERROR] Ошибка получения события %s: %v", eventID, err)
		return
	}

	timestamp, _ := strconv.ParseInt(event.Timestamp, 10, 64)
	modifiedTime := time.Unix(timestamp, 0).Format("02.01.2006 15:04:05")

	commentTime := "Неизвестно"
	if comment.Timestamp != "" {
		ts, _ := strconv.ParseInt(comment.Timestamp, 10, 64)
		commentTime = time.Unix(ts, 0).Format("02.01.2006 15:04:05")
	}

	creatorOrg := event.Orgc.Name
	if creatorOrg == "" {
		creatorOrg = "Неизвестно"
	}

	commentText := comment.Value
	if len(commentText) > 500 {
		commentText = commentText[:500] + "..."
	}

	message := fmt.Sprintf(`
*💬 НОВЫЙ КОММЕНТАРИЙ В СОБЫТИИ MISP*

*ID события:* %s
*Кем создано событие:* %s
*Название события:* %s
*Когда событие было создано:* %s
*Дата последнего редактирования:* %s
*IOCs:* %s

*Комментарий:*
%s

*Время комментария:* %s
`,
		event.ID,
		escapeMarkdown(creatorOrg),
		escapeMarkdown(event.Info),
		event.Date,
		modifiedTime,
		event.AttributeCount,
		escapeMarkdown(commentText),
		commentTime,
	)

	if err := m.bot.SendMessage(message); err != nil {
		m.logger.Errorf("[ERROR] Ошибка отправки уведомления о комментарии: %v", err)
	}
}

// processEvent обрабатывает событие и определяет тип обновления
func (m *Monitor) processEvent(event misp.Event) (isNew bool, isUpdated bool) {
	m.mu.Lock()
	existingState, exists := m.knownEvents[event.ID]
	m.mu.Unlock()

	if !exists {
		// Новое событие
		m.handleNewEvent(event)
		return true, false
	}

	// Проверяем изменения в существующем событии
	if event.Timestamp != existingState.LastTimestamp {
		m.handleEventUpdate(event, existingState)
		return false, true
	}

	return false, false
}

// handleNewEvent обрабатывает новое событие
func (m *Monitor) handleNewEvent(event misp.Event) {
	m.logger.Infof("[NEW EVENT] ID=%s | %s", event.ID, truncateString(event.Info, 50))

	// Получаем детальную информацию о событии
	eventDetail, err := m.client.GetEvent(event.ID)
	if err != nil {
		m.logger.Errorf("[ERROR] Ошибка получения деталей события %s: %v", event.ID, err)
		return
	}

	// Формируем и отправляем уведомление
	message := m.formatNewEventMessage(eventDetail)
	if err := m.bot.SendMessage(message); err != nil {
		m.logger.Errorf("[ERROR] Ошибка отправки уведомления: %v", err)
	} else {
		m.logger.Infof("[SENT] Уведомление о новом событии #%s отправлено", event.ID)
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
	m.knownComments[event.ID] = make(map[string]bool)
	m.mu.Unlock()
}

// handleEventUpdate обрабатывает обновление события
func (m *Monitor) handleEventUpdate(event misp.Event, oldState *EventState) {
	m.logger.Infof("[UPDATE EVENT] ID=%s | Обнаружено изменение", event.ID)

	// Получаем детальную информацию
	eventDetail, err := m.client.GetEvent(event.ID)
	if err != nil {
		m.logger.Errorf("[ERROR] Ошибка получения деталей события %s: %v", event.ID, err)
		return
	}

	// Определяем тип изменения
	newAttrCount, _ := strconv.Atoi(event.AttributeCount)

	var message string
	if newAttrCount > oldState.AttributeCount {
		addedCount := newAttrCount - oldState.AttributeCount
		m.logger.Infof("[UPDATE EVENT] Добавлено %d новых атрибутов", addedCount)
		message = m.formatEventUpdateMessage(eventDetail, oldState, addedCount)
	} else {
		m.logger.Info("[UPDATE EVENT] Общее обновление события")
		message = m.formatEventModifiedMessage(eventDetail)
	}

	if err := m.bot.SendMessage(message); err != nil {
		m.logger.Errorf("[ERROR] Ошибка отправки уведомления: %v", err)
	} else {
		m.logger.Infof("[SENT] Уведомление об обновлении события #%s отправлено", event.ID)
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
