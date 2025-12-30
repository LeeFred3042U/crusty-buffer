package worker

import (
	"context"
	"time"

	"crusty-buffer/internal/model"
	"crusty-buffer/internal/store"

	"github.com/go-shiori/go-readability"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Scraper defines the interface for downloading web pages.
// This allows us to mock the "Download" step in tests.
type Scraper interface {
	Scrape(url string, timeout time.Duration) (*readability.Article, error)
}

// DefaultScraper is the real implementation that uses the internet
type DefaultScraper struct{}

func (s *DefaultScraper) Scrape(url string, timeout time.Duration) (*readability.Article, error) {
	// We return a pointer to the article
	art, err := readability.FromURL(url, timeout)
	return &art, err
}

type Worker struct {
	store   store.Store
	logger  *zap.Logger
	scraper Scraper 
}

// NewWorker initializes the worker with the DefaultScraper
func NewWorker(store store.Store, logger *zap.Logger) *Worker {
	return &Worker{
		store:   store,
		logger:  logger,
		scraper: &DefaultScraper{},
	}
}

// Start runs the worker loop
func (w *Worker) Start(ctx context.Context) {
	w.logger.Info("Worker started. Waiting for jobs...")

	for {
		// Wait for job (Blocking call to Redis)
		id, err := w.store.PopQueue(ctx)
		if err != nil {
			if ctx.Err() != nil {
				w.logger.Info("Worker shutting down")
				return
			}
			w.logger.Error("Queue error", zap.Error(err))
			time.Sleep(time.Second)
			continue
		}

		// Process
		w.processJob(ctx, id)
	}
}

func (w *Worker) processJob(ctx context.Context, id uuid.UUID) {
	logger := w.logger.With(zap.String("job_id", id.String()))
	logger.Info("Processing started")

	// Fetch the Pending Article
	article, err := w.store.Get(ctx, id)
	if err != nil {
		logger.Error("Job failed: Article not found", zap.Error(err))
		return
	}

	// Download & Scrape (Using the Interface)
	logger.Info("Downloading", zap.String("url", article.URL))

	parsedArticle, err := w.scraper.Scrape(article.URL, 30*time.Second)
	if err != nil {
		logger.Error("Scraping failed", zap.Error(err))
		w.failJob(ctx, article, err.Error())
		return
	}

	// Update Article
	article.Title = parsedArticle.Title
	article.Content = parsedArticle.Content
	article.Excerpt = parsedArticle.Excerpt
	article.Status = model.StatusArchived
	now := time.Now()
	article.ArchivedAt = &now

	// Save the result
	if err := w.store.Save(ctx, article); err != nil {
		logger.Error("Failed to save result", zap.Error(err))
		return
	}

	logger.Info("Archiving complete", zap.String("title", article.Title))
}

func (w *Worker) failJob(ctx context.Context, article *model.Article, msg string) {
	article.Status = model.StatusFailed
	article.ErrorMessage = msg
	w.store.Save(ctx, article)
}