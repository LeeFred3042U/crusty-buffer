package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"http-streamer/handler"
	"http-streamer/storage"
)

var addr = flag.String("addr", ":3000", "http server address")

func main() {
	flag.Parse()

	// Initialize the database in a folder named "badger"
	storage.InitDB("badger")
	defer storage.CloseDB()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nShutting down server...")
		storage.CloseDB()
		os.Exit(0)
	}()

	router := handler.NewRouter()

	fmt.Println("Server listening on", *addr)
	fmt.Println("Open http://localhost:3000 in your browser.")
	log.Fatal(http.ListenAndServe(*addr, router))
}