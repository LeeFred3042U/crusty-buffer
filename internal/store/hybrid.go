package store

import (
	"context"
	"encoding/json"
	"fmt"

	"crusty-buffer/internal/model"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// HybridStore combines Redis (speed/queue) and Badger (heavy storage)
type HybridStore struct {
	rdb *redis.Client
	db  *badger.DB
}

// NewHybridStore initializes databases. 
// Pass badgerPath="" to run in "Redis-Only" mode (for CLI tools).
func NewHybridStore(redisAddr string, badgerPath string) (*HybridStore, error) {
	// Initialize Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	// Initialize Badger
	var db *badger.DB
	var err error
	
	if badgerPath != "" {
		opts := badger.DefaultOptions(badgerPath)
		opts.Logger = nil // Silence default logger
		db, err = badger.Open(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to open badger: %w", err)
		}
	}

	return &HybridStore{rdb: rdb, db: db}, nil
}

// Close cleans up connections
func (s *HybridStore) Close() {
	if s.rdb != nil {
		s.rdb.Close()
	}
	if s.db != nil {
		s.db.Close()
	}
}

// Save combines data: Metadata to Redis + Content to Badger
func (s *HybridStore) Save(ctx context.Context, article *model.Article) error {
	meta := *article
	meta.Content = "" 

	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	// Save Metadata to Redis Hash
	key := fmt.Sprintf("article:%s", article.ID)
	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, key, data, 0)

	// If it's a new pending article, add to Queue and Recent List
	if article.Status == model.StatusPending {
		pipe.LPush(ctx, "queue:archive", article.ID.String())
		pipe.LPush(ctx, "list:recent", article.ID.String())
		pipe.LTrim(ctx, "list:recent", 0, 49) // Keep only last 50 items
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	// If we have heavy content (HTML), save to Badger
	if article.Content != "" {
		if s.db == nil {
			// This happens if 'crusty add' tries to save content (which it shouldn't)
			// or if server is running in no-disk mode.
			return fmt.Errorf("cannot save content: badgerdb is not initialized")
		}
		err = s.db.Update(func(txn *badger.Txn) error {
			return txn.Set([]byte(article.ID.String()), []byte(article.Content))
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// Get combines data: Metadata from Redis + Content from Badger
func (s *HybridStore) Get(ctx context.Context, id uuid.UUID) (*model.Article, error) {
	// Fetch Metadata from Redis
	val, err := s.rdb.Get(ctx, fmt.Sprintf("article:%s", id)).Bytes()
	if err == redis.Nil {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	var article model.Article
	if err := json.Unmarshal(val, &article); err != nil {
		return nil, err
	}

	// Fetch Content from Badger (if available AND configured)
	if s.db != nil {
		err = s.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte(id.String()))
			if err != nil {
				return err
			}
			return item.Value(func(val []byte) error {
				article.Content = string(val)
				return nil
			})
		})

		if err != nil && err != badger.ErrKeyNotFound {
			return nil, err
		}
	}

	return &article, nil
}

// List fetches the most recent articles from Redis
func (s *HybridStore) List(ctx context.Context, limit int) ([]model.Article, error) {
	ids, err := s.rdb.LRange(ctx, "list:recent", 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}

	var articles []model.Article
	for _, idStr := range ids {
		val, err := s.rdb.Get(ctx, fmt.Sprintf("article:%s", idStr)).Bytes()
		if err == redis.Nil {
			continue
		}
		
		var a model.Article
		if err := json.Unmarshal(val, &a); err == nil {
			articles = append(articles, a)
		}
	}

	return articles, nil
}

// UpdateStatus is a helper to just flip the status flag in Redis
func (s *HybridStore) UpdateStatus(ctx context.Context, id uuid.UUID, status model.ArticleStatus) error {
	val, err := s.rdb.Get(ctx, fmt.Sprintf("article:%s", id)).Bytes()
	if err != nil {
		return err
	}

	var article model.Article
	if err := json.Unmarshal(val, &article); err != nil {
		return err
	}
	
	article.Status = status
	return s.Save(ctx, &article)
}

// PopQueue waits for a job in the Redis queue (Blocking)
func (s *HybridStore) PopQueue(ctx context.Context) (uuid.UUID, error) {
	// 0 means wait forever until an item arrives
	result, err := s.rdb.BRPop(ctx, 0, "queue:archive").Result()
	if err != nil {
		return uuid.Nil, err
	}

	idStr := result[1]
	return uuid.Parse(idStr)
}