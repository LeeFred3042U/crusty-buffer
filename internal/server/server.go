package web

import (
	"context"
	"html/template"
	"net/http"
	"time"

	"crusty-buffer/internal/model"
	"crusty-buffer/internal/store"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type Server struct {
	store  store.Store
	logger *zap.Logger
	router *mux.Router
	server *http.Server
}

func NewServer(st store.Store, logger *zap.Logger) *Server {
	s := &Server{
		store:  st,
		logger: logger,
		router: mux.NewRouter(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	// Static Files (CSS)
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// App Routes
	s.router.HandleFunc("/", s.handleIndex).Methods("GET")
	s.router.HandleFunc("/add", s.handleAdd).Methods("POST")
	s.router.HandleFunc("/view/{id}", s.handleView).Methods("GET")
}

// Start launches the HTTP server
func (s *Server) Start(port string) error {
	s.server = &http.Server{
		Addr:         ":" + port,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	s.logger.Info("Web server listening", zap.String("addr", port))
	return s.server.ListenAndServe()
}

// Stop gracefully shuts down
func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}


func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Fetch recent articles
	articles, err := s.store.List(r.Context(), 50)
	if err != nil {
		s.logger.Error("Failed to list articles", zap.Error(err))
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Render Template
	tmpl, err := template.ParseFiles("templates/layout.html", "templates/index.html", "templates/partials/archive_card.html")
	if err != nil {
		s.logger.Error("Template error", zap.Error(err))
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Articles": articles,
	}
	tmpl.Execute(w, data)
}

func (s *Server) handleView(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Fetch Article Content
	article, err := s.store.Get(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Render Template
	// Note: We use template.HTML to trust the content (since we stripped bad tags already)
	tmpl, err := template.ParseFiles("templates/layout.html", "templates/view.html")
	if err != nil {
		s.logger.Error("Template error", zap.Error(err))
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":      article.Title,
		"Content":    template.HTML(article.Content), 
		"OriginalURL": article.URL,
		"Date":       article.CreatedAt.Format("Jan 02, 2006"),
	}
	tmpl.Execute(w, data)
}

func (s *Server) handleAdd(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	if url == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	article := model.NewArticle(url)
	if err := s.store.Save(r.Context(), &article); err != nil {
		s.logger.Error("Failed to queue article", zap.Error(err))
		http.Error(w, "Failed to save", http.StatusInternalServerError)
		return
	}

	// Redirect back home
	http.Redirect(w, r, "/", http.StatusSeeOther)
}