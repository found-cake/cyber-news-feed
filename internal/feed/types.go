package feed

type Item struct {
	Title          string
	URL            string
	PublishedAt    *string
	PublishedRaw   string
	Categories     []string
	Description    string
	ContentEncoded string
	FeedID         string
	Authors        []Author
	Media          []Media
	SourceMetadata SourceMetadata
}

type Author struct {
	Name  string
	URI   string
	Email string
}

type Media struct {
	URL    string
	Kind   string
	Medium string
}

type SourceMetadata struct {
	GUIDIsPermalink string
	Image           SourceImage
	PostID          string
}

type SourceImage struct {
	URL    string
	Title  string
	Link   string
	Width  string
	Height string
}
