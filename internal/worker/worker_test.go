package worker

import (
	"context"
	"testing"
	"time"
	"fmt"

	"crusty-buffer/internal/model"
	"crusty-buffer/internal/store"

	"github.com/alicebob/miniredis/v2"
	"github.com/dgraph-io/badger/v4"
	"github.com/go-shiori/go-readability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type MockScraper struct {
	MockTitle   string
	MockContent string
	ShouldFail  bool
}

// Scrape simulates article scraping
func (m *MockScraper) Scrape(url string, timeout time.Duration) (*readability.Article, error) {
	if m.ShouldFail {
		return nil, fmt.Errorf("simulated 404 error")
	}
	return &readability.Article{
		Title:   m.MockTitle,
		Content: m.MockContent,
		Excerpt: "A short summary",
	}, nil
}

// TestWorker_ProcessJob tests that the worker correctly processes a job
// by fetching the article and updating its status
func TestWorker_ProcessJob(t *testing.T) {
	// Spin up fake Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// In-memory Badger so nothing touches disk
	opts := badger.DefaultOptions("").WithInMemory(true)
	opts.Logger = nil
	db, _ := badger.Open(opts)
	defer db.Close()

	// Real store wired to fake Redis + temp Badger
	st, err := store.NewHybridStore(mr.Addr(), t.TempDir())
	require.NoError(t, err)
	defer st.Close()

	// Worker with mocked scraper (no network, no flakiness)
	logger := zap.NewNop()
	w := NewWorker(st, logger)
	w.scraper = &MockScraper{
		MockTitle:   "Mocked Title",
		MockContent: "<p>This is fake content</p>",
	}

	// Seed a pending article
	article := model.NewArticle("http://fake-url.com")
	err = st.Save(context.Background(), &article)
	require.NoError(t, err)

	// Run worker asynchronously
	ctx, cancel := context.WithCancel(context.Background())
	go w.Start(ctx)

	// Give it time to process exactly one job
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Verify the article was processed and archived
	updatedArticle, err := st.Get(context.Background(), article.ID)
	require.NoError(t, err)

	assert.Equal(t, model.StatusArchived, updatedArticle.Status)
	assert.Equal(t, "Mocked Title", updatedArticle.Title)
	assert.Equal(t, "<p>This is fake content</p>", updatedArticle.Content)
}

// TestWorker_HandlesScrapeFailure tests that the worker correctly handles
// a scraping failure by updating the article status to "Failed"
func TestWorker_HandlesScrapeFailure(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	st, _ := store.NewHybridStore(mr.Addr(), t.TempDir())
	defer st.Close()

	// Setup Worker with a BROKEN Scraper
	logger := zap.NewNop()
	w := NewWorker(st, logger)
	w.scraper = &MockScraper{
		ShouldFail: true, // This will cause Scrape() to error
	}

	// Create a Job
	article := model.NewArticle("http://bad-url.com")
	st.Save(context.Background(), &article)

	// Run Worker briefly
	ctx, cancel := context.WithCancel(context.Background())
	go w.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Verify the Article Status is "Failed"
	savedArticle, err := st.Get(context.Background(), article.ID)
	require.NoError(t, err)

	assert.Equal(t, model.StatusFailed, savedArticle.Status)
	assert.Equal(t, "simulated 404 error", savedArticle.ErrorMessage)
}