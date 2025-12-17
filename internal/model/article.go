package model

import (
	"time"

	"github.com/google/uuid"
)

type ArticleStatus string

const (
	StatusPending  ArticleStatus = "pending"
	StatusArchived ArticleStatus = "archived"
	StatusFailed   ArticleStatus = "failed"
)


// Article represents a web article to be archived.
type Article struct {
	ID           uuid.UUID     `json:"id"`
	URL          string        `json:"url"`
	Title        string        `json:"title"`
	Excerpt      string        `json:"excerpt"`
	Content      string        `json:"content,omitempty"`
	Status       ArticleStatus `json:"status"`
	CreatedAt    time.Time     `json:"created_at"`
	ArchivedAt   *time.Time    `json:"archived_at,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
}

// NewArticle creates a new Article instance with the given URL and default values.
func NewArticle(rawURL string) Article {
	return Article{
		ID:        uuid.New(),
		URL:       rawURL,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}
}