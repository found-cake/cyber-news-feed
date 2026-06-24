package rssjson

import "encoding/json"

const SchemaVersion = 1

type Document struct {
	SchemaVersion int       `json:"schema_version"`
	Source        string    `json:"source"`
	UpdatedAt     string    `json:"updated_at"`
	RetentionDays int       `json:"retention_days"`
	Status        Status    `json:"status"`
	Articles      []Article `json:"articles"`
}

func (d *Document) UnmarshalJSON(data []byte) error {
	var raw documentJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for i := range raw.Articles {
		raw.Articles[i].SourceMetadata = raw.Articles[i].SourceMetadata.forSource(raw.Source, raw.Articles[i].legacyContent)
		raw.Articles[i].legacyContent = ""
	}
	*d = Document{
		SchemaVersion: raw.SchemaVersion,
		Source:        raw.Source,
		UpdatedAt:     raw.UpdatedAt,
		RetentionDays: raw.RetentionDays,
		Status:        raw.Status,
		Articles:      raw.Articles,
	}
	return nil
}

type Status struct {
	OK            bool    `json:"ok"`
	LastSuccessAt *string `json:"last_success_at"`
	LastErrorAt   *string `json:"last_error_at"`
	LastError     *string `json:"last_error"`
}

type Article struct {
	ID             string         `json:"id"`
	URL            string         `json:"url"`
	Title          string         `json:"title"`
	PublishedAt    *string        `json:"published_at"`
	PublishedRaw   string         `json:"published_raw"`
	Categories     []string       `json:"categories"`
	Description    string         `json:"description"`
	FeedID         string         `json:"feed_id"`
	Authors        []Author       `json:"authors"`
	Media          []Media        `json:"media"`
	SourceMetadata SourceMetadata `json:"source_metadata"`
	legacyContent  string
}

type Author struct {
	Name  string `json:"name"`
	URI   string `json:"uri"`
	Email string `json:"email"`
}

type Media struct {
	URL    string `json:"url"`
	Kind   string `json:"kind"`
	Medium string `json:"medium"`
}

type SourceMetadata struct {
	BoanNews          *BoanNewsMetadata          `json:"boannews,omitempty"`
	TheHackerNews     *TheHackerNewsMetadata     `json:"thehackernews,omitempty"`
	CybersecurityNews *CybersecurityNewsMetadata `json:"cybersecuritynews,omitempty"`
	StepSecurity      *StepSecurityMetadata      `json:"stepsecurity,omitempty"`
	DarkReading       *DarkReadingMetadata       `json:"darkreading,omitempty"`
	BleepingComputer  *BleepingComputerMetadata  `json:"bleepingcomputer,omitempty"`
	SecurityWeek      *SecurityWeekMetadata      `json:"securityweek,omitempty"`
	legacyGUID        string
	legacyPostID      string
}

type BoanNewsMetadata struct{}

type TheHackerNewsMetadata struct{}

type CybersecurityNewsMetadata struct {
	GUIDIsPermalink string `json:"guid_is_permalink"`
	PostID          string `json:"post_id"`
	ContentEncoded  string `json:"content_encoded"`
}

type StepSecurityMetadata struct{}

type DarkReadingMetadata struct {
	GUIDIsPermalink string `json:"guid_is_permalink"`
}

type BleepingComputerMetadata struct {
	GUIDIsPermalink string `json:"guid_is_permalink"`
}

type SecurityWeekMetadata struct {
	Image SecurityWeekImage `json:"image"`
}

type SecurityWeekImage struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Link   string `json:"link"`
	Width  string `json:"width"`
	Height string `json:"height"`
}

func (m *SourceMetadata) UnmarshalJSON(data []byte) error {
	var raw sourceMetadataJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*m = SourceMetadata{
		BoanNews:          raw.BoanNews,
		TheHackerNews:     raw.TheHackerNews,
		CybersecurityNews: raw.CybersecurityNews,
		StepSecurity:      raw.StepSecurity,
		DarkReading:       raw.DarkReading,
		BleepingComputer:  raw.BleepingComputer,
		SecurityWeek:      raw.SecurityWeek,
		legacyGUID:        raw.GUIDIsPermalink,
		legacyPostID:      raw.PostID,
	}
	return nil
}

func (m SourceMetadata) forSource(source string, contentEncoded string) SourceMetadata {
	switch source {
	case "cybersecuritynews":
		if m.CybersecurityNews == nil && (m.legacyGUID != "" || m.legacyPostID != "") {
			m.CybersecurityNews = &CybersecurityNewsMetadata{
				GUIDIsPermalink: m.legacyGUID,
				PostID:          m.legacyPostID,
			}
		}
		if m.CybersecurityNews == nil && contentEncoded != "" {
			m.CybersecurityNews = &CybersecurityNewsMetadata{}
		}
		if m.CybersecurityNews != nil && m.CybersecurityNews.ContentEncoded == "" {
			m.CybersecurityNews.ContentEncoded = contentEncoded
		}
	case "darkreading":
		if m.DarkReading == nil && m.legacyGUID != "" {
			m.DarkReading = &DarkReadingMetadata{GUIDIsPermalink: m.legacyGUID}
		}
	case "bleepingcomputer":
		if m.BleepingComputer == nil && m.legacyGUID != "" {
			m.BleepingComputer = &BleepingComputerMetadata{GUIDIsPermalink: m.legacyGUID}
		}
	}
	m.legacyGUID = ""
	m.legacyPostID = ""
	return m
}

func (a *Article) UnmarshalJSON(data []byte) error {
	var raw articleJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	categories := raw.Categories
	if len(categories) == 0 && raw.Category != "" {
		categories = []string{raw.Category}
	}
	*a = Article{
		ID:             raw.ID,
		URL:            raw.URL,
		Title:          raw.Title,
		PublishedAt:    raw.PublishedAt,
		PublishedRaw:   raw.PublishedRaw,
		Categories:     categories,
		Description:    raw.Description,
		FeedID:         raw.FeedID,
		Authors:        raw.Authors,
		Media:          raw.Media,
		SourceMetadata: raw.SourceMetadata,
		legacyContent:  raw.ContentEncoded,
	}
	return nil
}

type articleJSON struct {
	ID             string         `json:"id"`
	URL            string         `json:"url"`
	Title          string         `json:"title"`
	PublishedAt    *string        `json:"published_at"`
	PublishedRaw   string         `json:"published_raw"`
	Category       string         `json:"category"`
	Categories     []string       `json:"categories"`
	Description    string         `json:"description"`
	ContentEncoded string         `json:"content_encoded"`
	FeedID         string         `json:"feed_id"`
	Authors        []Author       `json:"authors"`
	Media          []Media        `json:"media"`
	SourceMetadata SourceMetadata `json:"source_metadata"`
}

type documentJSON struct {
	SchemaVersion int       `json:"schema_version"`
	Source        string    `json:"source"`
	UpdatedAt     string    `json:"updated_at"`
	RetentionDays int       `json:"retention_days"`
	Status        Status    `json:"status"`
	Articles      []Article `json:"articles"`
}

type sourceMetadataJSON struct {
	BoanNews          *BoanNewsMetadata          `json:"boannews"`
	TheHackerNews     *TheHackerNewsMetadata     `json:"thehackernews"`
	CybersecurityNews *CybersecurityNewsMetadata `json:"cybersecuritynews"`
	StepSecurity      *StepSecurityMetadata      `json:"stepsecurity"`
	DarkReading       *DarkReadingMetadata       `json:"darkreading"`
	BleepingComputer  *BleepingComputerMetadata  `json:"bleepingcomputer"`
	SecurityWeek      *SecurityWeekMetadata      `json:"securityweek"`
	GUIDIsPermalink   string                     `json:"guid_is_permalink"`
	PostID            string                     `json:"post_id"`
}
