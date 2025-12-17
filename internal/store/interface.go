package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"crusty-buffer/internal/model"
)

var (
	ErrNotFound = errors.New("article not found")
)

type Store interface {
	Save(ctx context.Context, article *model.Article) error
	Get(ctx context.Context, id uuid.UUID) (*model.Article, error)
	List(ctx context.Context, limit int) ([]model.Article, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.ArticleStatus) error
}