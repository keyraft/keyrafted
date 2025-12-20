package cmd

import (
	"context"
	"fmt"
	"keyrafted/internal/api"
	"keyrafted/internal/audit"
	"keyrafted/internal/auth"
	"keyrafted/internal/crypto"
	"keyrafted/internal/engine"
	"keyrafted/internal/storage"
	"keyrafted/internal/watch"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Keyraft server",
	Long:  "Start the Keyraft server with the specified configuration",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runStart(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	},
}

func runStart() error {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "keyraft.db")

	// Initialize storage
	store := storage.NewBoltDBStorage(dbPath)
	if err := store.Open(); err != nil {
		return fmt.Errorf("failed to open storage: %w", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			log.Printf("Error closing storage: %v", err)
		}
	}()

	// Initialize encryptor
	encryptor, err := crypto.NewEncryptorFromEnv("KEYRAFT_MASTER_KEY", masterKeyFile)
	if err != nil {
		return fmt.Errorf("failed to initialize encryptor: %w", err)
	}

	// Initialize services
	eng := engine.NewEngine(store, encryptor)
	authSvc := auth.NewService(store)
	watchMgr := watch.NewManager()
	auditSvc := audit.NewService(store)

	// Check if root token exists
	tokens, err := authSvc.ListTokens()
	if err != nil {
		return fmt.Errorf("failed to check tokens: %w", err)
	}
	if len(tokens) == 0 {
		log.Println("No tokens found. Run 'keyrafted init' to create the root token.")
		return fmt.Errorf("no authentication tokens configured")
	}

	// Create and start API server
	server := api.NewServer(listenAddr, eng, authSvc, watchMgr, auditSvc)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Received shutdown signal")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
		os.Exit(0)
	}()

	// Start server
	log.Printf("Keyraft server starting on %s", listenAddr)
	log.Printf("Data directory: %s", dataDir)
	if err := server.Start(); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(startCmd)

	// Start command flags
	startCmd.Flags().StringVar(&masterKey, "master-key", "", "Master encryption key (or use KEYRAFT_MASTER_KEY env var)")
	startCmd.Flags().StringVar(&masterKeyFile, "master-key-file", "", "Path to master encryption key file")
}
