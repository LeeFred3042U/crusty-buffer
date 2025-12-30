package store

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"crusty-buffer/internal/model"

	"github.com/alicebob/miniredis/v2"
	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHybridStore_Save_And_Get(t *testing.T) {
	// 1. Setup Mock Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// 2. Setup In-Memory Badger (Use "" as path and InMemory=true)
	opts := badger.DefaultOptions("").WithInMemory(true)
	opts.Logger = nil
	badgerDB, err := badger.Open(opts)
	require.NoError(t, err)
	defer badgerDB.Close()

	// 3. Initialize the Hybrid Store DIRECTLY
	// We skip NewHybridStore() to avoid creating real temp files for Badger.
	// Since we are in package 'store', we can set private fields 'rdb' and 'db'.
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := &HybridStore{
		rdb: rdb,
		db:  badgerDB,
	}
	defer store.Close()

	ctx := context.Background()

	// 4. Create a Dummy Article
	id := uuid.New()
	article := model.Article{
		ID:        id,
		URL:       "https://example.com",
		Title:     "Test Article",
		Content:   "<html><body><h1>Big Content</h1></body></html>",
		Status:    model.StatusPending,
		CreatedAt: time.Now(),
	}

	// 5. TEST: Save
	err = store.Save(ctx, &article)
	assert.NoError(t, err)

	// 6. VERIFY: Redis (Queue & Metadata)
	// Check Queue using Miniredis direct inspection
	// 'List' returns all elements in the list key
	queue, _ := mr.List("queue:archive")
	assert.Equal(t, 1, len(queue), "Should have 1 item in queue")
	assert.Equal(t, id.String(), queue[0], "Queue should contain the article ID")

	// Check Metadata (Title)
	val, err := mr.Get("article:" + id.String())
	assert.NoError(t, err, "Should find article metadata in Redis")
	
	var savedMeta model.Article
	json.Unmarshal([]byte(val), &savedMeta)
	assert.Equal(t, "Test Article", savedMeta.Title)
	assert.Empty(t, savedMeta.Content, "Redis should NOT store the heavy content")

	// 7. VERIFY: Badger (Content)
	err = badgerDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(id.String()))
		if err != nil {
			return err
		}
		val, _ := item.ValueCopy(nil)
		assert.Equal(t, article.Content, string(val), "Badger SHOULD have the heavy content")
		return nil
	})
	assert.NoError(t, err)
}

func TestHybridStore_ClientMode_NoBadger(t *testing.T) {
	// Setup Redis only (No Badger)
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// Initialize with EMPTY badger path (Simulating 'crusty add')
	store, err := NewHybridStore(mr.Addr(), "")
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Test: Saving Metadata (Should Succeed)
	// This is what 'crusty add' does: creates an article with URL but no content
	article := model.NewArticle("http://example.com")
	err = store.Save(ctx, &article)
	assert.NoError(t, err, "Saving metadata in client mode should work")

	// Verify it's in Redis
	exists := mr.Exists("article:" + article.ID.String())
	assert.True(t, exists, "Article metadata should be in Redis")

	// Test: Saving Content (Should Fail)
	// If we try to save HTML without Badger, it should block us
	article.Content = "<h1>Heavy HTML</h1>"
	err = store.Save(ctx, &article)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "badgerdb is not initialized", "Should prevent saving content without disk storage")
}