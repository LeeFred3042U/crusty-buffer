package handler

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// Upgrader configures the parameters for 
// upgrading an HTTP connection to a WebSocket connection
var upgrader = websocket.Upgrader{
	// CheckOrigin allows us to accept connections from any origin
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocketHandler handles the WebSocket connection requests
func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	log.Println("Client connected to WebSocket")

	// This is "echo" server loop
	for {
		// Read a message from the client
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		// Print the received message to the server's console
		log.Printf("Received message: %s", p)

		// Write the same message back to the client
		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Println("Write error:", err)
			break
		}
	}
}