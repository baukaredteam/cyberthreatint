package phishing

import (
	"strings"
	"unicode"

	"github.com/texttheater/golang-levenshtein/levenshtein"
	"phishing-monitor/pkg/models"
)

type Detector struct {
	clientDomains       []string
	phishingKeywords    []string
	similarityThreshold float64
}

func NewDetector(clientDomains []string, threshold float64) *Detector {
	// Ключевые слова, часто используемые в фишинге
	phishingKeywords := []string{
		"login", "secure", "account", "verify", "update", "bank",
		"payment", "confirm", "service", "support", "official",
		"security", "activate", "suspended", "urgent", "limited",
	}

	return &Detector{
		clientDomains:       clientDomains,
		phishingKeywords:    phishingKeywords,
		similarityThreshold: threshold,
	}
}

// AnalyzeDomain анализирует домен на фишинг
func (d *Detector) AnalyzeDomain(domain string) *models.SuspiciousDomain {
	domain = strings.ToLower(strings.TrimSpace(domain))
	
	// Пропускаем некорректные домены
	if !isValidDomain(domain) {
		return nil
	}

	for _, clientDomain := range d.clientDomains {
		similarity := d.calculateSimilarity(domain, clientDomain)
		
		if similarity >= d.similarityThreshold {
			confidence := d.calculateConfidence(domain, clientDomain, similarity)
			
			return &models.SuspiciousDomain{
				Domain:       domain,
				ClientDomain: clientDomain,
				Similarity:   similarity,
				Source:       "certstream",
				Confidence:   confidence,
				IsPhishing:   similarity > 0.9 || d.containsPhishingKeywords(domain),
				Notes:        d.generateNotes(domain, clientDomain, similarity),
			}
		}
	}

	return nil
}

// calculateSimilarity вычисляет схожесть доменов
func (d *Detector) calculateSimilarity(domain1, domain2 string) float64 {
	// Убираем TLD для лучшего сравнения
	base1 := extractBaseDomain(domain1)
	base2 := extractBaseDomain(domain2)

	// Расстояние Левенштейна
	distance := levenshtein.DistanceForStrings([]rune(base1), []rune(base2), levenshtein.DefaultOptions)
	maxLen := max(len(base1), len(base2))
	
	if maxLen == 0 {
		return 0
	}

	similarity := 1.0 - float64(distance)/float64(maxLen)
	
	// Дополнительные проверки
	if containsSubstring(base1, base2) || containsSubstring(base2, base1) {
		similarity += 0.2
	}

	if similarity > 1.0 {
		similarity = 1.0
	}

	return similarity
}

// calculateConfidence определяет уровень уверенности
func (d *Detector) calculateConfidence(domain, clientDomain string, similarity float64) string {
	if similarity > 0.95 {
		return "high"
	} else if similarity > 0.85 {
		return "medium"
	}
	return "low"
}

// containsPhishingKeywords проверяет наличие фишинговых ключевых слов
func (d *Detector) containsPhishingKeywords(domain string) bool {
	for _, keyword := range d.phishingKeywords {
		if strings.Contains(domain, keyword) {
			return true
		}
	}
	return false
}

// generateNotes создает заметки об анализе
func (d *Detector) generateNotes(domain, clientDomain string, similarity float64) string {
	notes := ""
	
	if similarity > 0.9 {
		notes += "Очень высокая схожесть с клиентским доменом. "
	}
	
	if d.containsPhishingKeywords(domain) {
		notes += "Содержит фишинговые ключевые слова. "
	}
	
	if isIDN(domain) {
		notes += "Содержит интернационализированные символы (возможный Punycode атака). "
	}
	
	return strings.TrimSpace(notes)
}

// extractBaseDomain извлекает базовое имя домена без TLD
func extractBaseDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) > 1 {
		return parts[0]
	}
	return domain
}

// containsSubstring проверяет содержание подстроки
func containsSubstring(s1, s2 string) bool {
	return strings.Contains(s1, s2) || strings.Contains(s2, s1)
}

// isValidDomain проверяет корректность домена
func isValidDomain(domain string) bool {
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}
	
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}
	
	return strings.Contains(domain, ".")
}

// isIDN проверяет наличие международных символов
func isIDN(domain string) bool {
	for _, r := range domain {
		if r > unicode.MaxASCII {
			return true
		}
	}
	return false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}