package updater

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"phishing-monitor/pkg/storage"
)

type SourceUpdater struct {
	storage *storage.Storage
	sources []string
}

func NewSourceUpdater(storage *storage.Storage) *SourceUpdater {
	sources := []string{
		"https://raw.githubusercontent.com/xRuffKez/NRD/main/lists/30-day_dga/domains-only/nrd-30day-dga_part1.txt",
		"https://raw.githubusercontent.com/Phishing-Database/Phishing.Database/master/phishing-domains/domains.txt",
		"https://raw.githubusercontent.com/hagezi/dns-blocklists/main/domains/pro.txt",
	}

	return &SourceUpdater{
		storage: storage,
		sources: sources,
	}
}

func (u *SourceUpdater) UpdateAllSources() error {
	logrus.Info("Начинаем обновление баз данных фишинга...")
	
	totalUpdated := 0
	for i, source := range u.sources {
		logrus.Infof("Обновляем источник %d/%d: %s", i+1, len(u.sources), source)
		
		count, err := u.updateFromSource(source)
		if err != nil {
			logrus.Errorf("Ошибка обновления источника %s: %v", source, err)
			continue
		}
		
		totalUpdated += count
		logrus.Infof("Обновлено %d доменов из источника %s", count, source)
		
		// Пауза между запросами
		time.Sleep(2 * time.Second)
	}
	
	logrus.Infof("Обновление завершено. Всего обновлено: %d доменов", totalUpdated)
	return nil
}

func (u *SourceUpdater) updateFromSource(sourceURL string) (int, error) {
	resp, err := http.Get(sourceURL)
	if err != nil {
		return 0, fmt.Errorf("не удалось загрузить %s: %w", sourceURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP ошибка %d для %s", resp.StatusCode, sourceURL)
	}

	return u.processDomainList(resp.Body, sourceURL)
}

func (u *SourceUpdater) processDomainList(reader io.Reader, source string) (int, error) {
	scanner := bufio.NewScanner(reader)
	count := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Пропускаем комментарии и пустые строки
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		// Извлекаем домен (может быть в формате "domain" или "0.0.0.0 domain")
		domain := extractDomain(line)
		if domain == "" || !isValidDomain(domain) {
			continue
		}

		// Сохраняем в базу данных
		if err := u.storage.SavePhishingDomain(domain, getSourceName(source)); err != nil {
			// Игнорируем ошибки дублирования
			if !strings.Contains(err.Error(), "UNIQUE") {
				logrus.Debugf("Ошибка сохранения домена %s: %v", domain, err)
			}
			continue
		}

		count++
	}

	if err := scanner.Err(); err != nil {
		return count, fmt.Errorf("ошибка чтения данных: %w", err)
	}

	return count, nil
}

func extractDomain(line string) string {
	// Обрабатываем различные форматы:
	// 1. domain.com
	// 2. 0.0.0.0 domain.com
	// 3. ||domain.com^
	// 4. *.domain.com

	line = strings.TrimSpace(line)
	
	// Удаляем AdBlock синтаксис
	if strings.HasPrefix(line, "||") && strings.HasSuffix(line, "^") {
		line = line[2 : len(line)-1]
	}
	
	// Разделяем по пробелам и берем последнюю часть (для hosts формата)
	parts := strings.Fields(line)
	if len(parts) > 1 {
		line = parts[len(parts)-1]
	}
	
	// Убираем wildcard
	if strings.HasPrefix(line, "*.") {
		line = line[2:]
	}
	
	// Убираем протокол если есть
	if strings.HasPrefix(line, "http://") {
		line = line[7:]
	}
	if strings.HasPrefix(line, "https://") {
		line = line[8:]
	}
	
	// Убираем путь если есть
	if idx := strings.Index(line, "/"); idx != -1 {
		line = line[:idx]
	}
	
	return strings.ToLower(line)
}

func getSourceName(url string) string {
	if strings.Contains(url, "xRuffKez/NRD") {
		return "NRD"
	} else if strings.Contains(url, "Phishing-Database") {
		return "Phishing-Database"
	} else if strings.Contains(url, "hagezi") {
		return "Hagezi"
	}
	return "Unknown"
}

func isValidDomain(domain string) bool {
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}
	
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}
	
	if !strings.Contains(domain, ".") {
		return false
	}
	
	// Проверяем что это не IP адрес
	parts := strings.Split(domain, ".")
	if len(parts) == 4 {
		allNumbers := true
		for _, part := range parts {
			if len(part) == 0 || len(part) > 3 {
				allNumbers = false
				break
			}
			for _, char := range part {
				if char < '0' || char > '9' {
					allNumbers = false
					break
				}
			}
		}
		if allNumbers {
			return false // Это IP адрес
		}
	}
	
	return true
}