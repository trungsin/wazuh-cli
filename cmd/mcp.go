package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ba0f3/wazuh-cli/internal/client"
	"github.com/ba0f3/wazuh-cli/internal/config"
	"github.com/ba0f3/wazuh-cli/internal/indexer"
	mcpserver "github.com/ba0f3/wazuh-cli/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server (stdio transport)",
	Long: `Start a Model Context Protocol (MCP) server over stdio.

The MCP server exposes Wazuh API operations as tools that AI agents
(Claude Desktop, Cursor, Cline) can call directly. Connect by adding
this to your MCP client config:

  {
    "mcpServers": {
      "wazuh": {
        "command": "wazuh-cli",
        "args": ["mcp"]
      }
    }
  }

Configuration is read from the same sources as other commands:
CLI flags, WAZUH_* env vars, .env file, or ~/.config/wazuh/config.json.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configPath, &flagOverrides)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		if err := cfg.Validate(); err != nil {
			return err
		}

		apiClient, err := client.NewClient(cfg)
		if err != nil {
			return fmt.Errorf("creating API client: %w", err)
		}

		var idxClient *indexer.Client
		if cfg.IndexerURL != "" {
			c, err := indexer.NewClient(cfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "WARNING: indexer client init failed: %v (alert tools disabled)\n", err)
			} else {
				idxClient = c
			}
		}

		opts := mcpserver.ServerOpts{
			Client:       apiClient,
			Indexer:      idxClient,
			ReadOnly:     cfg.MCPReadOnly,
			Quiet:        cfg.Quiet,
			Version:      Version,
			IndexerIndex: cfg.IndexerIndex,
		}

		server := mcpserver.NewServer(opts)

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		return server.Run(ctx)
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
