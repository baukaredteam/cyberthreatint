package misp

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client представляет клиент для работы с MISP API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient создает новый MISP клиент
func NewClient(baseURL, apiKey string, skipTLSVerify bool) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipTLSVerify,
		},
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

// doRequest выполняет HTTP запрос к MISP API
func (c *Client) doRequest(method, endpoint string, body io.Reader) ([]byte, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Authorization", c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ошибка API (код %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// GetEvents получает список событий
func (c *Client) GetEvents() ([]Event, error) {
	data, err := c.doRequest("GET", "/events/index", nil)
	if err != nil {
		return nil, err
	}

	var events []Event
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("ошибка парсинга событий: %w", err)
	}

	return events, nil
}

// GetEvent получает детальную информацию о событии по ID
func (c *Client) GetEvent(eventID string) (*EventDetail, error) {
	data, err := c.doRequest("GET", fmt.Sprintf("/events/view/%s", eventID), nil)
	if err != nil {
		return nil, err
	}

	var response EventDetailResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("ошибка парсинга события: %w", err)
	}

	return &response.Event, nil
}

// GetRecentEvents получает события за последние N минут
func (c *Client) GetRecentEvents(minutes int) ([]Event, error) {
	timestamp := time.Now().Add(-time.Duration(minutes) * time.Minute).Unix()
	endpoint := fmt.Sprintf("/events/index/timestamp:%d", timestamp)

	data, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var events []Event
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("ошибка парсинга событий: %w", err)
	}

	return events, nil
}

// GetModifiedEvents получает события, измененные после указанного timestamp
func (c *Client) GetModifiedEvents(sinceTimestamp int64) ([]Event, error) {
	endpoint := fmt.Sprintf("/events/index/timestamp:%d", sinceTimestamp)

	data, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var events []Event
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("ошибка парсинга событий: %w", err)
	}

	return events, nil
}

// GetEventComments получает комментарии события через атрибуты типа "comment"
func (c *Client) GetEventComments(eventID string) ([]Attribute, error) {
	event, err := c.GetEvent(eventID)
	if err != nil {
		return nil, err
	}

	var comments []Attribute
	for _, attr := range event.Attributes {
		if attr.Type == "comment" || attr.Category == "Internal reference" {
			comments = append(comments, attr)
		}
	}

	return comments, nil
}

// GetOrganisation получает информацию об организации
func (c *Client) GetOrganisation(orgID string) (*Organisation, error) {
	data, err := c.doRequest("GET", fmt.Sprintf("/organisations/view/%s", orgID), nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Organisation Organisation `json:"Organisation"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("ошибка парсинга организации: %w", err)
	}

	return &response.Organisation, nil
}

// SearchEvents ищет события по параметрам
func (c *Client) SearchEvents(params map[string]interface{}) ([]Event, error) {
	// Для поиска используем POST запрос
	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации параметров: %w", err)
	}

	data, err := c.doRequest("POST", "/events/restSearch", io.NopCloser(
		&jsonReader{data: jsonData},
	))
	if err != nil {
		return nil, err
	}

	var response struct {
		Response []EventDetailResponse `json:"response"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("ошибка парсинга результатов поиска: %w", err)
	}

	events := make([]Event, 0, len(response.Response))
	for _, r := range response.Response {
		events = append(events, Event{
			ID:            r.Event.ID,
			OrgID:         r.Event.OrgID,
			Info:          r.Event.Info,
			Date:          r.Event.Date,
			Timestamp:     r.Event.Timestamp,
			Published:     r.Event.Published,
			AttributeCount: r.Event.AttributeCount,
			Distribution:  r.Event.Distribution,
			Org:           r.Event.Org,
		})
	}

	return events, nil
}

type jsonReader struct {
	data []byte
	pos  int
}

func (r *jsonReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
