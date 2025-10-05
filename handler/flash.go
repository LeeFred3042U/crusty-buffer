package handler

import (
	"net/http"
	"sync"
)

var (
	flashMessages = make(map[string]string)
	flashMutex    = &sync.Mutex{}
)

// setFlash sets a message for a user session using remote address
func setFlash(_ http.ResponseWriter, r *http.Request, message string) {
	flashMutex.Lock()
	defer flashMutex.Unlock()
	flashMessages[r.RemoteAddr] = message
}

// getFlash retrieves and immediately deletes a message
func getFlash(_ http.ResponseWriter, r *http.Request) string {
	flashMutex.Lock()
	defer flashMutex.Unlock()
	message, ok := flashMessages[r.RemoteAddr]
	if ok {
		delete(flashMessages, r.RemoteAddr)
		return message
	}
	return ""
}