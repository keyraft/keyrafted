package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "keyrafted",
	Short: "Keyrafted is the server component of the Keyraft project — a lightweight, self-hosted, distributed configuration and secrets management system.",
	Long: `Keyrafted is responsible for running the cluster nodes, managing replication,
storing configuration and secrets securely, and serving requests from clients, CLI, and SDKs.

Features:
  • Distributed key-value store with Raft-based replication
  • Encrypted storage (AES-256-GCM) for secrets
  • Namespaces for isolation
  • Token-based authentication with scoped API keys
  • Versioned configuration values
  • Watch API for live updates
  • HTTP/JSON protocol

Example:
  # Start single node
  keyrafted --data-dir /data --listen :7331

  # Start multi-node cluster
  keyrafted --data-dir /data1 --listen :7331 --node-id node1 --peers node2:7332,node3:7333`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.keyrafted.yaml)")
}
