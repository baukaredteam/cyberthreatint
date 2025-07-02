package certstream

import (
	"fmt"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type Client struct {
	url        string
	conn       *websocket.Conn
	domainChan chan string
	stopChan   chan struct{}
}

type CertStreamMessage struct {
	MessageType string `json:"message_type"`
	Data        struct {
		UpdateType string `json:"update_type"`
		LeafCert   struct {
			Subject struct {
				CN string `json:"CN"`
			} `json:"subject"`
			Extensions struct {
				SubjectAltName []string `json:"subjectAltName"`
			} `json:"extensions"`
		} `json:"leaf_cert"`
	} `json:"data"`
}

func NewClient(url string) *Client {
	return &Client{
		url:        url,
		domainChan: make(chan string, 1000),
		stopChan:   make(chan struct{}),
	}
}

func (c *Client) Connect() error {
	var err error
	c.conn, _, err = websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return fmt.Errorf("не удалось подключиться к certstream: %w", err)
	}

	logrus.Info("Подключение к certstream установлено")
	return nil
}

func (c *Client) Start() {
	go c.listen()
}

func (c *Client) listen() {
	defer func() {
		if c.conn != nil {
			c.conn.Close()
		}
	}()

	for {
		select {
		case <-c.stopChan:
			return
		default:
			var message CertStreamMessage
			err := c.conn.ReadJSON(&message)
			if err != nil {
				logrus.Errorf("Ошибка чтения сообщения: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			c.processCertificate(message)
		}
	}
}

func (c *Client) processCertificate(message CertStreamMessage) {
	if message.MessageType != "certificate_update" {
		return
	}

	domains := c.extractDomains(message)
	for _, domain := range domains {
		select {
		case c.domainChan <- domain:
		default:
			// Канал переполнен, пропускаем
		}
	}
}

func (c *Client) extractDomains(message CertStreamMessage) []string {
	var domains []string

	// Извлекаем CN (Common Name)
	if cn := message.Data.LeafCert.Subject.CN; cn != "" {
		cn = strings.ToLower(strings.TrimSpace(cn))
		if isValidDomain(cn) {
			domains = append(domains, cn)
		}
	}

	// Извлекаем SAN (Subject Alternative Names)
	for _, san := range message.Data.LeafCert.Extensions.SubjectAltName {
		san = strings.ToLower(strings.TrimSpace(san))
		if strings.HasPrefix(san, "dns:") {
			domain := strings.TrimPrefix(san, "dns:")
			if isValidDomain(domain) {
				domains = append(domains, domain)
			}
		}
	}

	return removeDuplicates(domains)
}

func (c *Client) GetDomainChannel() <-chan string {
	return c.domainChan
}

func (c *Client) Stop() {
	close(c.stopChan)
	if c.conn != nil {
		c.conn.Close()
	}
	close(c.domainChan)
}

func isValidDomain(domain string) bool {
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}
	
	if strings.HasPrefix(domain, "*.") {
		domain = domain[2:] // Убираем wildcard
	}
	
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}
	
	return strings.Contains(domain, ".")
}

func removeDuplicates(domains []string) []string {
	keys := make(map[string]bool)
	var result []string
	
	for _, domain := range domains {
		if !keys[domain] {
			keys[domain] = true
			result = append(result, domain)
		}
	}
	
	return result
}