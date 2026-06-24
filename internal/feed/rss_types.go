package feed

import "encoding/xml"

type envelope struct {
	XMLName xml.Name
	Channel rssChannel  `xml:"channel"`
	Entries []atomEntry `xml:"entry"`
}

type rssChannel struct {
	Image rssImage  `xml:"image"`
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string      `xml:"title"`
	Links       []rssLink   `xml:"link"`
	GUID        rssGUID     `xml:"guid"`
	PubDate     string      `xml:"pubDate"`
	DCDate      string      `xml:"date"`
	Published   string      `xml:"published"`
	Updated     string      `xml:"updated"`
	Categories  []string    `xml:"category"`
	Description string      `xml:"description"`
	Content     string      `xml:"encoded"`
	Creators    []string    `xml:"creator"`
	Authors     []rssAuthor `xml:"author"`
	Thumbnails  []mediaNode `xml:"thumbnail"`
	Media       []mediaNode `xml:"content"`
	Image       rssImage    `xml:"image"`
	PostID      string      `xml:"post-id"`
}

type rssLink struct {
	Href string `xml:"href,attr"`
	Text string `xml:",chardata"`
}

type rssGUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Text        string `xml:",chardata"`
}

type rssAuthor struct {
	Text string `xml:",chardata"`
}

type rssImage struct {
	Href   string `xml:"href,attr"`
	URL    string `xml:"url"`
	Title  string `xml:"title"`
	Link   string `xml:"link"`
	Width  string `xml:"width"`
	Height string `xml:"height"`
	Text   string `xml:",chardata"`
}

type atomEntry struct {
	ID         string         `xml:"id"`
	Title      string         `xml:"title"`
	Links      []atomLink     `xml:"link"`
	Published  string         `xml:"published"`
	Updated    string         `xml:"updated"`
	Categories []atomCategory `xml:"category"`
	Summary    string         `xml:"summary"`
	Content    string         `xml:"content"`
	Authors    []atomAuthor   `xml:"author"`
	Thumbnails []mediaNode    `xml:"thumbnail"`
}

type atomLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
	Text string `xml:",chardata"`
}

type atomCategory struct {
	Term string `xml:"term,attr"`
	Text string `xml:",chardata"`
}

type atomAuthor struct {
	Name  string `xml:"name"`
	URI   string `xml:"uri"`
	Email string `xml:"email"`
}

type mediaNode struct {
	URL    string `xml:"url,attr"`
	Medium string `xml:"medium,attr"`
}
