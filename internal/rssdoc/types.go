package rssdoc

const SchemaVersion = 1

type Document struct {
	SchemaVersion int       `json:"schema_version"`
	Source        string    `json:"source"`
	UpdatedAt     string    `json:"updated_at"`
	RetentionDays int       `json:"retention_days"`
	Status        Status    `json:"status"`
	Articles      []Article `json:"articles"`
}

type Status struct {
	OK            bool    `json:"ok"`
	LastSuccessAt *string `json:"last_success_at"`
	LastErrorAt   *string `json:"last_error_at"`
	LastError     *string `json:"last_error"`
}

type Article struct {
	ID           string  `json:"id"`
	URL          string  `json:"url"`
	Title        string  `json:"title"`
	PublishedAt  *string `json:"published_at"`
	PublishedRaw string  `json:"published_raw"`
	Category     string  `json:"category"`
}
