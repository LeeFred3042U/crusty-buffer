package handler

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"sort"

	"http-streamer/storage"
	"http-streamer/models"

	"github.com/dgraph-io/badger/v4"
)

// Home serves the main index page
func Home(w http.ResponseWriter, r *http.Request) {
	var articles []models.ArticleMetadata
	err := storage.DB.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("meta:")
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var meta models.ArticleMetadata
				if err := json.Unmarshal(val, &meta); err != nil {
					return err
				}
				articles = append(articles, meta)
				return nil
			})
			if err != nil {
				log.Printf("Error processing metadata: %v", err)
			}
		}
		return nil
	})

	if err != nil {
		http.Error(w, "Failed to list archived items", http.StatusInternalServerError)
		return
	}

	// Sort articles by date
	sort.Slice(articles, func(i, j int) bool {
		return articles[i].ArchivedAt.After(articles[j].ArchivedAt)
	})

	// Convert data models to view models for the template
	archivesForView := make([]models.ArchiveViewModel, len(articles))
	for i, article := range articles {
		archivesForView[i] = models.NewArchiveViewModel(article)
	}

	// Parse all required template files
	tmpl, err := template.ParseFiles(
		"templates/layout.html", 
		"templates/index.html", 
		"templates/partials/archive_card.html",
	)
	if err != nil {
		http.Error(w, "Failed to parse templates", http.StatusInternalServerError)
		return
	}

	// Data for the template
	data := struct {
		Title    string
		Archives []models.ArchiveViewModel
		Flash    string
	}{
		Title:    "Dashboard",
		Archives: archivesForView,
		Flash:    getFlash(w, r),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = tmpl.ExecuteTemplate(w, "layout", data)
	if err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
	}
}