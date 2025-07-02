package models

import (
	"time"
)

type SuspiciousDomain struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Domain       string    `gorm:"uniqueIndex" json:"domain"`
	ClientDomain string    `json:"client_domain"`
	Similarity   float64   `json:"similarity"`
	Source       string    `json:"source"` // certstream, nrd, phishing-db
	Confidence   string    `json:"confidence"` // low, medium, high
	IsPhishing   bool      `json:"is_phishing"`
	Notes        string    `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PhishingDatabase struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Domain    string    `gorm:"uniqueIndex" json:"domain"`
	Source    string    `json:"source"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ClientDomain struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Domain   string `gorm:"uniqueIndex" json:"domain"`
	Company  string `json:"company"`
	IsActive bool   `json:"is_active"`
}

func (SuspiciousDomain) TableName() string {
	return "suspicious_domains"
}

func (PhishingDatabase) TableName() string {
	return "phishing_database"
}

func (ClientDomain) TableName() string {
	return "client_domains"
}