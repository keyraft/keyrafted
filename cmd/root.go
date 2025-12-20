package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	dataDir       string
	listenAddr    string
	masterKey     string
	masterKeyFile string
)

var rootCmd = &cobra.Command{
	Use:   "keyrafted",
	Short: "Keyrafted is the server component of the Keyraft project",
	Long: `Keyrafted - A lightweight, self-hosted configuration and secrets management system.

Features:
  • Key-value store with versioning
  • Encrypted storage (AES-256-GCM) for secrets
  • Namespaces for isolation
  • Token-based authentication with scoped API keys
  • Watch API for live updates
  • HTTP/JSON protocol

Example:
  # Start server
  keyrafted start --data-dir /data --listen :7200

  # Initialize root token
  keyrafted init --data-dir /data`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Persistent flags
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", "./data", "Data directory for storage")
	rootCmd.PersistentFlags().StringVar(&listenAddr, "listen", ":7200", "HTTP listen address")
}
