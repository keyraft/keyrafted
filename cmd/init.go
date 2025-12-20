package cmd

import (
	"fmt"
	"keyrafted/internal/auth"
	"keyrafted/internal/storage"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Keyraft and generate root token",
	Long:  "Initialize the Keyraft database and generate the initial root authentication token",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runInit(); err != nil {
			log.Fatalf("Failed to initialize: %v", err)
		}
	},
}

func runInit() error {
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

	// Initialize auth service
	authSvc := auth.NewService(store)

	// Generate root token
	rootToken, err := authSvc.InitializeRootToken()
	if err != nil {
		return fmt.Errorf("failed to initialize root token: %w", err)
	}

	fmt.Println("✓ Keyraft initialized successfully!")
	fmt.Println("\nRoot token (save this securely):")
	fmt.Printf("\n  %s\n\n", rootToken.Token)
	fmt.Println("Use this token to authenticate API requests:")
	fmt.Printf("  curl -H 'Authorization: Bearer %s' http://localhost%s/v1/health\n\n", rootToken.Token, listenAddr)

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
}
