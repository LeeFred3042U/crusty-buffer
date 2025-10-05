package models

import (
	"net/url"
	"time"
)

// ArticleMetadata holds the raw data we store in the db
type ArticleMetadata struct {
	URL        string    `json:"url"`
	Title      string    `json:"title"`
	ArchivedAt time.Time `json:"archived_at"`
	ContentKey string    `json:"content_key"`
}

// ArchiveViewModel is a struct tailored for presentation in the templates
// It contains formatted data derived from ArticleMetadata
type ArchiveViewModel struct {
	Key       string
	Title     string
	Timestamp string
	Domain    string
}

// NewArchiveViewModel creates a view model from the raw metadata
func NewArchiveViewModel(meta ArticleMetadata) ArchiveViewModel {
	domain := "unknown"
	parsedURL, err := url.Parse(meta.URL)
	if err == nil {
		domain = parsedURL.Hostname()
	}

	return ArchiveViewModel{
		Key:       meta.ContentKey,
		Title:     meta.Title,
		Timestamp: meta.ArchivedAt.Format("02 Jan 2006, 15:04 MST"),
		Domain:    domain,
	}
}