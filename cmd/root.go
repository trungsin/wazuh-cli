// Package cmd contains all wazuh-cli Cobra commands.
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ba0f3/wazuh-cli/internal/client"
	"github.com/ba0f3/wazuh-cli/internal/config"
	"github.com/ba0f3/wazuh-cli/internal/output"
	"github.com/spf13/cobra"
)

// Version is the current version of wazuh-cli (set by ldflags).
var Version = "dev"

// flagOverrides holds values parsed from CLI flags (populated in PersistentPreRunE).
var flagOverrides config.Config

// globalCfg is the resolved configuration available to all subcommands.
var globalCfg *config.Config

// globalClient is the shared Wazuh API client.
var globalClient *client.Client

// globalFmt is the shared output formatter.
var globalFmt *output.Formatter

// configPath is the path to the JSON config file (overridden by --config).
var configPath string

var rootCmd = &cobra.Command{
	Use:   "wazuh-cli",
	Short: "AI-agent-first CLI for the Wazuh Server API",
	Long: `wazuh-cli — Wazuh Server API command-line interface

Supports all Wazuh API resources: agents, rules, decoders, vulnerabilities,
FIM, SCA, syscollector, active response, cluster management, and RBAC.

Configuration priority (highest → lowest):
  1. CLI flags
  2. Environment variables (WAZUH_URL, WAZUH_USER, WAZUH_PASSWORD, ...)
  3. .env file in current directory
  4. ~/.config/wazuh/config.json

Output: JSON (default), Markdown (--output markdown), or Raw (--output raw)
Errors: machine-readable JSON on stdout, human text on stderr`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip for commands that don't need auth (init, config, auth, version, completion)
		name := cmd.Name()
		if name == "init" || name == "version" || name == "completion" || name == "mcp" ||
			name == "config" || (cmd.HasParent() && cmd.Parent().Name() == "config") ||
			name == "auth" || (cmd.HasParent() && cmd.Parent().Name() == "auth") ||
			name == "alert" || (cmd.HasParent() && cmd.Parent().Name() == "alert") {
			return nil
		}

		// Handle --password - (read from stdin)
		if flagOverrides.Password == "-" {
			fmt.Fprint(os.Stderr, "Password: ")
			reader := bufio.NewReader(os.Stdin)
			pw, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading password from stdin: %w", err)
			}
			flagOverrides.Password = strings.TrimRight(pw, "\r\n")
		}

		// Resolve configuration
		cfg, err := config.Load(configPath, &flagOverrides)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// Validate (don't validate for init subcommand)
		if err := cfg.Validate(); err != nil {
			return err
		}

		globalCfg = cfg

		// Build client
		c, err := client.NewClient(cfg)
		if err != nil {
			return fmt.Errorf("creating API client: %w", err)
		}
		globalClient = c

		// Build formatter
		globalFmt = output.New(cfg.Output, cfg.Pretty)

		return nil
	},
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	pf := rootCmd.PersistentFlags()

	pf.StringVar(&configPath, "config", "", "path to config file (default: ~/.config/wazuh/config.json)")
	pf.StringVarP(&flagOverrides.URL, "url", "u", "", "Wazuh API URL (e.g. https://wazuh:55000)")
	pf.StringVar(&flagOverrides.User, "user", "", "API username")
	pf.StringVar(&flagOverrides.Password, "password", "", "API password (use '-' to read from stdin)")
	pf.StringVar(&flagOverrides.RawToken, "token", "", "use raw JWT token instead of user/password")
	pf.StringVarP(&flagOverrides.Output, "output", "o", "", "output format: json (default), markdown, raw")
	pf.BoolVar(&flagOverrides.Pretty, "pretty", false, "pretty-print JSON output")
	pf.BoolVarP(&flagOverrides.Insecure, "insecure", "k", false, "skip TLS certificate verification")
	pf.StringVar(&flagOverrides.CACert, "ca-cert", "", "path to custom CA certificate")
	pf.StringVar(&flagOverrides.ClientCert, "client-cert", "", "path to client certificate (mTLS)")
	pf.StringVar(&flagOverrides.ClientKey, "client-key", "", "path to client private key (mTLS)")
	pf.IntVar(&flagOverrides.Timeout, "timeout", 0, "HTTP timeout in seconds (default: 30)")
	pf.BoolVar(&flagOverrides.Debug, "debug", false, "enable debug logging to stderr")
	pf.BoolVarP(&flagOverrides.Quiet, "quiet", "q", false, "suppress informational messages")
}

// HandleError prints a formatted error and returns the appropriate exit code.
func HandleError(err error) int {
	if err == nil {
		return 0
	}
	code := output.ExitCode(err)
	if globalFmt != nil {
		globalFmt.WriteError(code, err.Error(), "")
	} else {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}
	return code
}

// mustWrite is a helper that writes output and exits on error.
func mustWrite(data interface{}) {
	if err := globalFmt.Write(data); err != nil {
		fmt.Fprintln(os.Stderr, "Error writing output:", err)
		os.Exit(1)
	}
}
