package storage

import (
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"phishing-monitor/pkg/models"
)

type Storage struct {
	db *gorm.DB
}

func NewStorage(dbPath string) (*Storage, error) {
	// Создаем директорию для базы данных если не существует
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	// Автомиграция
	if err := db.AutoMigrate(
		&models.SuspiciousDomain{},
		&models.PhishingDatabase{},
		&models.ClientDomain{},
	); err != nil {
		return nil, err
	}

	return &Storage{db: db}, nil
}

func (s *Storage) SaveSuspiciousDomain(domain *models.SuspiciousDomain) error {
	return s.db.Create(domain).Error
}

func (s *Storage) GetSuspiciousDomains(limit int) ([]models.SuspiciousDomain, error) {
	var domains []models.SuspiciousDomain
	err := s.db.Order("created_at DESC").Limit(limit).Find(&domains).Error
	return domains, err
}

func (s *Storage) DomainExists(domain string) bool {
	var count int64
	s.db.Model(&models.SuspiciousDomain{}).Where("domain = ?", domain).Count(&count)
	return count > 0
}

func (s *Storage) SavePhishingDomain(domain string, source string) error {
	phishing := &models.PhishingDatabase{
		Domain: domain,
		Source: source,
	}
	return s.db.Create(phishing).Error
}

func (s *Storage) IsKnownPhishingDomain(domain string) bool {
	var count int64
	s.db.Model(&models.PhishingDatabase{}).Where("domain = ?", domain).Count(&count)
	return count > 0
}

func (s *Storage) SaveClientDomain(domain, company string) error {
	client := &models.ClientDomain{
		Domain:   domain,
		Company:  company,
		IsActive: true,
	}
	return s.db.Create(client).Error
}

func (s *Storage) GetClientDomains() ([]models.ClientDomain, error) {
	var domains []models.ClientDomain
	err := s.db.Where("is_active = ?", true).Find(&domains).Error
	return domains, err
}

func (s *Storage) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}