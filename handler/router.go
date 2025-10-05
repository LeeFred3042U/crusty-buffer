package handler

import "net/http"

// NewRouter creates and configures a new server router
func NewRouter() *http.ServeMux {
    mux := http.NewServeMux()

    mux.HandleFunc("/", Home)
    mux.HandleFunc("/ws", WebSocketHandler)
    mux.HandleFunc("/archive", Archive)
    mux.HandleFunc("/view", View)

    // Serve static files
    staticHandler := http.FileServer(http.Dir("static"))
    mux.Handle("/static/", http.StripPrefix("/static/", staticHandler))

    return mux
}