package storage

import (
	"log"
	"time"

	"github.com/dgraph-io/badger/v4"
)

var DB *badger.DB

// InitDB initializes the BadgerDB database connection
func InitDB(path string) {
	var err error
	opts := badger.DefaultOptions(path)
	opts.Logger = nil 
	
	DB, err = badger.Open(opts)
	if err != nil {
		log.Fatalf("Failed to open BadgerDB: %v", err)
	}

	go func() {
		for {
			time.Sleep(5 * time.Minute)
			if err := DB.RunValueLogGC(0.7); err != nil {
				// add log error try logrus
            }
		}
	}()
}

// duh it closes the db
func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}