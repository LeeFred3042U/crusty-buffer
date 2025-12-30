package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"crusty-buffer/internal/model"
	"crusty-buffer/internal/store"
	"crusty-buffer/internal/worker"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	logger     *zap.Logger
	redisAddr  string
	badgerPath string
)

var rootCmd = &cobra.Command{
	Use:   "crusty",
	Short: "crusty-buffer - A self-hosted read-it-later tool",
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the worker and web server",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Setup Signal Handling (Ctrl+C)
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Setup Manual 'q' input handling
		go func() {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				if scanner.Text() == "q" {
					fmt.Println(" 'q' pressed. Stopping...")
					cancel()
					return
				}
			}
		}()

		// Handle shutdown signals
		go func() {
			<-sigChan
			logger.Info("Shutting down...")
			cancel()
		}()

		// Initialize Store (FULL MODE - Redis + Badger)
		st, err := store.NewHybridStore(redisAddr, badgerPath)
		if err != nil {
			logger.Fatal("Failed to init store", zap.Error(err))
		}
		defer st.Close()

		// Start Worker
		w := worker.NewWorker(st, logger)
		go w.Start(ctx)

		logger.Info("Server running.")
		fmt.Println("Press 'q' + Enter or Ctrl+C to stop.")
		
		// Block until shutdown
		<-ctx.Done()
		
		time.Sleep(1 * time.Second)
		logger.Info("Goodbye!")
	},
}

var addCmd = &cobra.Command{
	Use:   "add [url]",
	Short: "Archive a URL immediately",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]
		
		// Initialize Store (CLIENT MODE - Redis Only)
		// Passing "" as the second argument ensures we don't try to open the BadgerDB file lock.
		st, err := store.NewHybridStore(redisAddr, "") 
		if err != nil {
			logger.Fatal("Failed to init store", zap.Error(err))
		}
		defer st.Close()

		// Create Article
		article := model.NewArticle(url)

		// Save (Pushes to Redis Queue, ignores Badger because content is empty)
		ctx := context.Background()
		if err := st.Save(ctx, &article); err != nil {
			logger.Fatal("Failed to save article", zap.Error(err))
		}

		logger.Info("Article queued", 
			zap.String("id", article.ID.String()), 
			zap.String("url", url))
	},
}

func main() {
	var err error
	logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	rootCmd.PersistentFlags().StringVar(&redisAddr, "redis", "localhost:6379", "Address of Redis server")
	rootCmd.PersistentFlags().StringVar(&badgerPath, "badger", "./badger-data", "Path to BadgerDB data directory")

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(addCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}