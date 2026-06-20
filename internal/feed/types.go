package feed

type Item struct {
	Title        string
	URL          string
	PublishedAt  *string
	PublishedRaw string
	Categories   []string
	Summary      string
}
