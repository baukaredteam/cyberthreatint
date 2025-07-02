package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	"phishing-monitor/config"
	"phishing-monitor/pkg/certstream"
	"phishing-monitor/pkg/models"
	"phishing-monitor/pkg/phishing"
	"phishing-monitor/pkg/storage"
	"phishing-monitor/pkg/telegram"
	"phishing-monitor/pkg/updater"
)

type MonitorService struct {
	config       *config.Config
	storage      *storage.Storage
	detector     *phishing.Detector
	certClient   *certstream.Client
	telegramBot  *telegram.Bot
	updater      *updater.SourceUpdater
	cron         *cron.Cron
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

func main() {
	// Инициализация логирования
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Загрузка конфигурации
	cfg := config.Load()
	
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	logrus.Info("🚀 Запуск Phishing Monitor...")

	// Создание сервиса
	service, err := NewMonitorService(cfg)
	if err != nil {
		logrus.Fatalf("Ошибка создания сервиса: %v", err)
	}

	// Обработка сигналов для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logrus.Info("Получен сигнал завершения, останавливаем сервис...")
		cancel()
	}()

	// Запуск сервиса
	if err := service.Start(ctx); err != nil {
		logrus.Fatalf("Ошибка запуска сервиса: %v", err)
	}

	logrus.Info("👋 Phishing Monitor остановлен")
}

func NewMonitorService(cfg *config.Config) (*MonitorService, error) {
	// Инициализация хранилища
	storage, err := storage.NewStorage(cfg.DatabasePath)
	if err != nil {
		return nil, err
	}

	// Инициализация детектора фишинга
	detector := phishing.NewDetector(cfg.ClientDomains, cfg.SimilarityThreshold)

	// Инициализация Certstream клиента
	certClient := certstream.NewClient(cfg.CertstreamURL)

	// Инициализация Telegram бота
	telegramBot, err := telegram.NewBot(cfg.TelegramBotToken, cfg.TelegramChatID)
	if err != nil {
		return nil, err
	}

	// Инициализация обновлятора баз данных
	sourceUpdater := updater.NewSourceUpdater(storage)

	// Инициализация cron для задач по расписанию
	cronScheduler := cron.New(cron.WithSeconds())

	ctx, cancel := context.WithCancel(context.Background())

	return &MonitorService{
		config:      cfg,
		storage:     storage,
		detector:    detector,
		certClient:  certClient,
		telegramBot: telegramBot,
		updater:     sourceUpdater,
		cron:        cronScheduler,
		ctx:         ctx,
		cancel:      cancel,
	}, nil
}

func (s *MonitorService) Start(ctx context.Context) error {
	s.ctx = ctx

	// Первоначальное обновление баз данных
	logrus.Info("Выполняем первоначальное обновление баз данных...")
	if err := s.updater.UpdateAllSources(); err != nil {
		logrus.Errorf("Ошибка обновления баз данных: %v", err)
	}

	// Инициализация клиентских доменов в базе данных
	s.initializeClientDomains()

	// Подключение к Certstream
	if err := s.certClient.Connect(); err != nil {
		return err
	}

	// Настройка cron задач
	s.setupCronJobs()

	// Отправка уведомления о запуске
	s.telegramBot.SendStatusUpdate("🛡️ Phishing Monitor запущен и готов к мониторингу!")

	// Запуск горутин
	s.wg.Add(3)
	go s.runCertStreamMonitoring()
	go s.runTelegramBot()
	go s.runCronScheduler()

	// Ожидание завершения
	<-ctx.Done()
	return s.shutdown()
}

func (s *MonitorService) initializeClientDomains() {
	for _, domain := range s.config.ClientDomains {
		if err := s.storage.SaveClientDomain(domain, "Unknown"); err != nil {
			if !contains(err.Error(), "UNIQUE") {
				logrus.Debugf("Ошибка сохранения клиентского домена %s: %v", domain, err)
			}
		}
	}
}

func (s *MonitorService) setupCronJobs() {
	// Обновление баз данных каждые 6 часов
	s.cron.AddFunc("0 0 */6 * * *", func() {
		logrus.Info("Запуск периодического обновления баз данных...")
		if err := s.updater.UpdateAllSources(); err != nil {
			logrus.Errorf("Ошибка периодического обновления: %v", err)
		}
	})

	// Ежедневный отчет в 9:00
	s.cron.AddFunc("0 0 9 * * *", func() {
		s.sendDailyReport()
	})
}

func (s *MonitorService) runCertStreamMonitoring() {
	defer s.wg.Done()

	s.certClient.Start()
	domainChan := s.certClient.GetDomainChannel()

	for {
		select {
		case <-s.ctx.Done():
			s.certClient.Stop()
			return
		case domain := <-domainChan:
			s.processDomain(domain)
		}
	}
}

func (s *MonitorService) processDomain(domain string) {
	// Проверяем, не обрабатывали ли мы уже этот домен
	if s.storage.DomainExists(domain) {
		return
	}

	// Анализируем домен на фишинг
	suspiciousDomain := s.detector.AnalyzeDomain(domain)
	if suspiciousDomain == nil {
		return
	}

	// Проверяем в базе известных фишинговых доменов
	if s.storage.IsKnownPhishingDomain(domain) {
		suspiciousDomain.IsPhishing = true
		suspiciousDomain.Notes += " Домен найден в базе фишинговых доменов."
	}

	// Сохраняем в базу данных
	if err := s.storage.SaveSuspiciousDomain(suspiciousDomain); err != nil {
		logrus.Errorf("Ошибка сохранения подозрительного домена: %v", err)
		return
	}

	// Отправляем уведомление в Telegram
	if err := s.telegramBot.SendSuspiciousDomainAlert(suspiciousDomain); err != nil {
		logrus.Errorf("Ошибка отправки уведомления: %v", err)
	}

	logrus.Infof("Обнаружен подозрительный домен: %s (схожесть: %.2f%%)", 
		domain, suspiciousDomain.Similarity*100)
}

func (s *MonitorService) runTelegramBot() {
	defer s.wg.Done()
	// Запуск обработчика команд Telegram бота в отдельной горутине
	go s.telegramBot.StartCommandHandler()
	<-s.ctx.Done()
}

func (s *MonitorService) runCronScheduler() {
	defer s.wg.Done()
	s.cron.Start()
	<-s.ctx.Done()
	s.cron.Stop()
}

func (s *MonitorService) sendDailyReport() {
	// Получаем домены за последние 24 часа
	domains, err := s.storage.GetSuspiciousDomains(100)
	if err != nil {
		logrus.Errorf("Ошибка получения доменов для отчета: %v", err)
		return
	}

	// Фильтруем домены за последние 24 часа
	var recentDomains []models.SuspiciousDomain
	yesterday := time.Now().Add(-24 * time.Hour)

	for _, domain := range domains {
		if domain.CreatedAt.After(yesterday) {
			recentDomains = append(recentDomains, domain)
		}
	}

	if err := s.telegramBot.SendDailyReport(recentDomains); err != nil {
		logrus.Errorf("Ошибка отправки ежедневного отчета: %v", err)
	}
}

func (s *MonitorService) shutdown() error {
	logrus.Info("Завершение работы сервиса...")
	
	s.cancel()
	s.wg.Wait()
	
	if err := s.storage.Close(); err != nil {
		logrus.Errorf("Ошибка закрытия базы данных: %v", err)
		return err
	}

	logrus.Info("Сервис успешно остановлен")
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}