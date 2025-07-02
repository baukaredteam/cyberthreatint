package telegram

import (
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"phishing-monitor/pkg/models"
)

type Bot struct {
	api    *tgbotapi.BotAPI
	chatID int64
}

func NewBot(token string, chatID int64) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать Telegram бота: %w", err)
	}

	api.Debug = false
	logrus.Infof("Telegram бот авторизован: %s", api.Self.UserName)

	return &Bot{
		api:    api,
		chatID: chatID,
	}, nil
}

func (b *Bot) SendSuspiciousDomainAlert(domain *models.SuspiciousDomain) error {
	message := b.formatSuspiciousDomainMessage(domain)
	
	msg := tgbotapi.NewMessage(b.chatID, message)
	msg.ParseMode = "HTML"
	
	_, err := b.api.Send(msg)
	if err != nil {
		return fmt.Errorf("не удалось отправить сообщение: %w", err)
	}
	
	return nil
}

func (b *Bot) SendStatusUpdate(message string) error {
	msg := tgbotapi.NewMessage(b.chatID, message)
	_, err := b.api.Send(msg)
	return err
}

func (b *Bot) formatSuspiciousDomainMessage(domain *models.SuspiciousDomain) string {
	var builder strings.Builder
	
	// Заголовок с эмодзи в зависимости от уровня опасности
	var icon string
	switch domain.Confidence {
	case "high":
		icon = "🔴"
	case "medium":
		icon = "🟡"
	default:
		icon = "🟢"
	}
	
	builder.WriteString(fmt.Sprintf("%s <b>ПОДОЗРИТЕЛЬНЫЙ ДОМЕН ОБНАРУЖЕН</b>\n\n", icon))
	
	// Основная информация
	builder.WriteString(fmt.Sprintf("🎯 <b>Домен:</b> <code>%s</code>\n", domain.Domain))
	builder.WriteString(fmt.Sprintf("🏢 <b>Клиент:</b> %s\n", domain.ClientDomain))
	builder.WriteString(fmt.Sprintf("📊 <b>Схожесть:</b> %.2f%%\n", domain.Similarity*100))
	builder.WriteString(fmt.Sprintf("⚠️ <b>Уровень риска:</b> %s\n", strings.ToUpper(domain.Confidence)))
	builder.WriteString(fmt.Sprintf("📡 <b>Источник:</b> %s\n", domain.Source))
	
	// Флаг фишинга
	if domain.IsPhishing {
		builder.WriteString("🚨 <b>ФИШИНГ:</b> Да\n")
	} else {
		builder.WriteString("✅ <b>ФИШИНГ:</b> Не определен\n")
	}
	
	// Заметки
	if domain.Notes != "" {
		builder.WriteString(fmt.Sprintf("📝 <b>Заметки:</b> %s\n", domain.Notes))
	}
	
	// Время обнаружения
	builder.WriteString(fmt.Sprintf("🕒 <b>Время:</b> %s\n", 
		domain.CreatedAt.Format("02.01.2006 15:04:05")))
	
	// Рекомендации
	builder.WriteString("\n<b>Рекомендации:</b>\n")
	if domain.Confidence == "high" || domain.IsPhishing {
		builder.WriteString("• Немедленно проверить домен\n")
		builder.WriteString("• Заблокировать домен в DNS/Proxy\n")
		builder.WriteString("• Уведомить клиента\n")
	} else {
		builder.WriteString("• Проверить домен на предмет фишинга\n")
		builder.WriteString("• Добавить в список мониторинга\n")
	}
	
	// Ссылки для проверки
	builder.WriteString(fmt.Sprintf("\n<b>Проверка:</b>\n"))
	builder.WriteString(fmt.Sprintf("• <a href=\"https://www.virustotal.com/gui/domain/%s\">VirusTotal</a>\n", domain.Domain))
	builder.WriteString(fmt.Sprintf("• <a href=\"https://urlvoid.com/scan/%s\">URLVoid</a>\n", domain.Domain))
	builder.WriteString(fmt.Sprintf("• <a href=\"https://whois.net/whois/%s\">WHOIS</a>\n", domain.Domain))
	
	return builder.String()
}

func (b *Bot) SendDailyReport(domains []models.SuspiciousDomain) error {
	if len(domains) == 0 {
		message := "📊 <b>Ежедневный отчет</b>\n\n✅ За последние 24 часа подозрительных доменов не обнаружено."
		return b.SendStatusUpdate(message)
	}
	
	var builder strings.Builder
	builder.WriteString("📊 <b>Ежедневный отчет мониторинга</b>\n\n")
	builder.WriteString(fmt.Sprintf("📈 <b>Всего обнаружено:</b> %d доменов\n\n", len(domains)))
	
	// Группируем по клиентам
	clientStats := make(map[string]int)
	highRiskCount := 0
	
	for _, domain := range domains {
		clientStats[domain.ClientDomain]++
		if domain.Confidence == "high" || domain.IsPhishing {
			highRiskCount++
		}
	}
	
	builder.WriteString("<b>По клиентам:</b>\n")
	for client, count := range clientStats {
		builder.WriteString(fmt.Sprintf("• %s: %d доменов\n", client, count))
	}
	
	builder.WriteString(fmt.Sprintf("\n🚨 <b>Высокий риск:</b> %d доменов\n", highRiskCount))
	
	// Показываем последние 5 доменов с высоким риском
	if highRiskCount > 0 {
		builder.WriteString("\n<b>Последние домены высокого риска:</b>\n")
		count := 0
		for _, domain := range domains {
			if (domain.Confidence == "high" || domain.IsPhishing) && count < 5 {
				builder.WriteString(fmt.Sprintf("• <code>%s</code> (%.0f%%)\n", 
					domain.Domain, domain.Similarity*100))
				count++
			}
		}
	}
	
	msg := tgbotapi.NewMessage(b.chatID, builder.String())
	msg.ParseMode = "HTML"
	
	_, err := b.api.Send(msg)
	return err
}

func (b *Bot) StartCommandHandler() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		switch update.Message.Command() {
		case "start":
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, 
				"🛡️ Phishing Monitor активен!\n\nВы будете получать уведомления о подозрительных доменах.")
			b.api.Send(msg)
		case "status":
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, 
				fmt.Sprintf("✅ Мониторинг активен\n🕒 Время: %s", time.Now().Format("02.01.2006 15:04:05")))
			b.api.Send(msg)
		}
	}
}