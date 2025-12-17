package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var logger *zap.Logger

// rootCmd represents the base command when called without any subcommands
// Usage: crusty
var rootCmd = &cobra.Command{
	Use:   "crusty",
	Short: "Crusty Buffer - A self-hosted read-it-later tool",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// serverCmd represents the command to start the web server
// Usage: crusty server
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the web server",
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Server starting...")
		logger.Warn("Handler logic missing - waiting for implementation")
	},
}

// addCmd represents the command to add a URL for archiving
// Usage: crusty add [url]
var addCmd = &cobra.Command{
	Use:   "add [url]",
	Short: "Archive a URL immediately",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]
		logger.Info("Queueing URL for archive", zap.String("url", url))
		logger.Warn("Worker logic missing - waiting for implementation")
	},
}

func main() {
	var err error
	logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(addCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}