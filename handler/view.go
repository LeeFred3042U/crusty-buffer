package handler

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"

	"http-streamer/storage"
	"http-streamer/models"

	"github.com/dgraph-io/badger/v4"
)

// View handles the request to display an archived page
func View(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	var compressedContent []byte
	var metadataJSON []byte
	metaKey := "meta:" + key

	err := storage.DB.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(metaKey))
		if err != nil {
			return err
		}
		metadataJSON, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}

		item, err = txn.Get([]byte(key))
		if err != nil {
			return err
		}
		compressedContent, err = item.ValueCopy(nil)
		return err
	})

	if err != nil {
		if err == badger.ErrKeyNotFound {
			http.Error(w, "Archived page not found", http.StatusNotFound)
		} else {
			log.Printf("Failed to retrieve from BadgerDB: %v", err)
			http.Error(w, "Failed to retrieve content", http.StatusInternalServerError)
		}
		return
	}

	var metadata models.ArticleMetadata
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		http.Error(w, "Failed to parse metadata", http.StatusInternalServerError)
		return
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(compressedContent))
	if err != nil {
		http.Error(w, "Failed to create gzip reader", http.StatusInternalServerError)
		return
	}
	defer gzipReader.Close()

	decompressedContent, err := io.ReadAll(gzipReader)
	if err != nil {
		http.Error(w, "Failed to decompress content", http.StatusInternalServerError)
		return
	}

	// Parse the new template files.
	tmpl, err := template.ParseFiles("templates/layout.html", "templates/view.html")
	if err != nil {
		http.Error(w, "Failed to parse templates", http.StatusInternalServerError)
		return
	}

	// Data for the view template
	data := struct {
		Title       string
		OriginalURL string
		Content     template.HTML
	}{
		Title:       metadata.Title,
		OriginalURL: metadata.URL,
		Content:     template.HTML(decompressedContent),
	}

	// Execute the main layout template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = tmpl.ExecuteTemplate(w, "layout", data)
	if err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
	}
}