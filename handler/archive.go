package handler

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"http-streamer/storage"
	"http-streamer/models"

	"github.com/dgraph-io/badger/v4"
	"github.com/go-shiori/go-readability"
)

// Archive handles the request to archive a new URL.
func Archive(w http.ResponseWriter, r *http.Request) {
	rawURL := r.FormValue("url")
	if rawURL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	key := sha256.Sum256([]byte(rawURL))
	keyStr := hex.EncodeToString(key[:])
	metaKey := "meta:" + keyStr

	err := storage.DB.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(metaKey))
		return err
	})
	if err == nil {
		setFlash(w, r, fmt.Sprintf("URL already archived: %s", rawURL))
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	resp, err := http.Get(rawURL)
	if err != nil {
		setFlash(w, r, "Error: Failed to fetch URL.")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	defer resp.Body.Close()

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		setFlash(w, r, "Error: Invalid URL provided.")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	article, err := readability.FromReader(resp.Body, parsedURL)
	if err != nil {
		setFlash(w, r, "Error: Failed to parse article content.")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	var compressedContent bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressedContent)
	io.WriteString(gzipWriter, article.Content)
	gzipWriter.Close()

	metadata := models.ArticleMetadata{
		URL:        rawURL,
		Title:      article.Title,
		ArchivedAt: time.Now(),
		ContentKey: keyStr,
	}
	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		http.Error(w, "Failed to create metadata", http.StatusInternalServerError)
		return
	}

	err = storage.DB.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(keyStr), compressedContent.Bytes())
		if err := txn.SetEntry(e); err != nil {
			return err
		}
		e = badger.NewEntry([]byte(metaKey), jsonMetadata)
		return txn.SetEntry(e)
	})

	if err != nil {
		log.Printf("Failed to save to BadgerDB: %v", err)
		setFlash(w, r, "Error: Failed to save content to database.")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Set a success message and redirect.
	setFlash(w, r, fmt.Sprintf("Successfully archived '%s'", article.Title))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}