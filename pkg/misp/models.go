package misp

// Event представляет событие MISP
type Event struct {
	ID              string       `json:"id"`
	OrgID           string       `json:"org_id"`
	OrgcID          string       `json:"orgc_id"`
	Info            string       `json:"info"`
	UUID            string       `json:"uuid"`
	Date            string       `json:"date"`
	Published       bool         `json:"published"`
	Analysis        string       `json:"analysis"`
	Distribution    string       `json:"distribution"`
	Timestamp       string       `json:"timestamp"`
	PublishTimestamp string      `json:"publish_timestamp"`
	AttributeCount  string       `json:"attribute_count"`
	ThreatLevelID   string       `json:"threat_level_id"`
	Org             Organisation `json:"Org"`
	Orgc            Organisation `json:"Orgc"`
}

// EventDetail представляет детальную информацию о событии
type EventDetail struct {
	ID               string        `json:"id"`
	OrgID            string        `json:"org_id"`
	OrgcID           string        `json:"orgc_id"`
	Info             string        `json:"info"`
	UUID             string        `json:"uuid"`
	Date             string        `json:"date"`
	Published        bool          `json:"published"`
	Analysis         string        `json:"analysis"`
	Distribution     string        `json:"distribution"`
	Timestamp        string        `json:"timestamp"`
	PublishTimestamp string        `json:"publish_timestamp"`
	AttributeCount   string        `json:"attribute_count"`
	ThreatLevelID    string        `json:"threat_level_id"`
	Org              Organisation  `json:"Org"`
	Orgc             Organisation  `json:"Orgc"`
	Attributes       []Attribute   `json:"Attribute"`
	Tags             []Tag         `json:"Tag"`
	ShadowAttributes []interface{} `json:"ShadowAttribute"`
	RelatedEvents    []interface{} `json:"RelatedEvent"`
}

// EventDetailResponse обертка для ответа API
type EventDetailResponse struct {
	Event EventDetail `json:"Event"`
}

// Organisation представляет организацию в MISP
type Organisation struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	UUID        string `json:"uuid"`
	Description string `json:"description"`
	Nationality string `json:"nationality"`
	Sector      string `json:"sector"`
	Type        string `json:"type"`
	Local       bool   `json:"local"`
}

// Attribute представляет атрибут события
type Attribute struct {
	ID             string `json:"id"`
	EventID        string `json:"event_id"`
	ObjectID       string `json:"object_id"`
	ObjectRelation string `json:"object_relation"`
	Category       string `json:"category"`
	Type           string `json:"type"`
	Value          string `json:"value"`
	ToIDS          bool   `json:"to_ids"`
	UUID           string `json:"uuid"`
	Timestamp      string `json:"timestamp"`
	Distribution   string `json:"distribution"`
	Comment        string `json:"comment"`
	Deleted        bool   `json:"deleted"`
	DisableCorrelation bool `json:"disable_correlation"`
}

// Tag представляет тег события
type Tag struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Colour         string `json:"colour"`
	Exportable     bool   `json:"exportable"`
	OrgID          string `json:"org_id"`
	UserID         string `json:"user_id"`
	HideTag        bool   `json:"hide_tag"`
	NumericalValue string `json:"numerical_value"`
	IsGalaxy       bool   `json:"is_galaxy"`
	IsCustomGalaxy bool   `json:"is_custom_galaxy"`
	LocalOnly      bool   `json:"local_only"`
}

// SightingType представляет типы Sighting в MISP
type SightingType int

const (
	SightingTypeSighting    SightingType = 0 // Обнаружен
	SightingTypeFalsePositive SightingType = 1 // Ложное срабатывание
	SightingTypeExpiration  SightingType = 2 // Истёк срок
)

// Post представляет пост/комментарий к событию (Event Thread)
type Post struct {
	ID         string `json:"id"`
	DateCreated string `json:"date_created"`
	DateModified string `json:"date_modified"`
	UserID     string `json:"user_id"`
	Contents   string `json:"contents"`
	PostID     string `json:"post_id"` // Parent post ID for replies
	ThreadID   string `json:"thread_id"`
	User       User   `json:"User"`
}

// Thread представляет тред обсуждения события
type Thread struct {
	ID             string `json:"id"`
	DateCreated    string `json:"date_created"`
	DateModified   string `json:"date_modified"`
	Distribution   string `json:"distribution"`
	EventID        string `json:"event_id"`
	OrgID          string `json:"org_id"`
	SharingGroupID string `json:"sharing_group_id"`
	Title          string `json:"title"`
	Posts          []Post `json:"Post"`
}

// User представляет пользователя MISP
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	OrgID string `json:"org_id"`
}

// DistributionLevel уровни распределения
type DistributionLevel int

const (
	DistributionOrganisationOnly DistributionLevel = 0
	DistributionCommunityOnly    DistributionLevel = 1
	DistributionConnectedCommunities DistributionLevel = 2
	DistributionAllCommunities   DistributionLevel = 3
	DistributionSharingGroup     DistributionLevel = 4
	DistributionInherit          DistributionLevel = 5
)

// DistributionName возвращает название уровня распределения
func DistributionName(level string) string {
	switch level {
	case "0":
		return "Your organisation only"
	case "1":
		return "This community only"
	case "2":
		return "Connected communities"
	case "3":
		return "All communities"
	case "4":
		return "Sharing group"
	case "5":
		return "Inherit event"
	default:
		return "Unknown"
	}
}

// ThreatLevelName возвращает название уровня угрозы
func ThreatLevelName(level string) string {
	switch level {
	case "1":
		return "High"
	case "2":
		return "Medium"
	case "3":
		return "Low"
	case "4":
		return "Undefined"
	default:
		return "Unknown"
	}
}

// AnalysisLevelName возвращает название уровня анализа
func AnalysisLevelName(level string) string {
	switch level {
	case "0":
		return "Initial"
	case "1":
		return "Ongoing"
	case "2":
		return "Complete"
	default:
		return "Unknown"
	}
}
