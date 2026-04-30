// Package config resolves the wazuh-cli configuration from multiple sources
// in priority order: CLI flags > environment variables > .env file > config file.
package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// DefaultConfigDir is the XDG-compliant config directory.
	DefaultConfigDir = ".config/wazuh"
	// DefaultConfigFile is the config file name inside DefaultConfigDir.
	DefaultConfigFile = "config.json"
	// DefaultTokenFile is the JWT token cache file name.
	DefaultTokenFile = "token"
	// DefaultTimeout is the default HTTP timeout in seconds.
	DefaultTimeout = 30
	// DefaultOutput is the default output format.
	DefaultOutput = "json"
)

// Config holds all wazuh-cli runtime configuration.
type Config struct {
	// Connection
	URL      string `json:"url"`
	User     string `json:"user"`
	Password string `json:"password"`

	// TLS
	Insecure   bool   `json:"insecure"`
	CACert     string `json:"ca_cert"`
	ClientCert string `json:"client_cert"`
	ClientKey  string `json:"client_key"`

	// Indexer (OpenSearch)
	IndexerURL      string `json:"indexer_url"`
	IndexerUser     string `json:"indexer_user"`
	IndexerPassword string `json:"indexer_password"`
	IndexerIndex    string `json:"indexer_index"`

	// Behavior
	Timeout int    `json:"timeout"`
	Output  string `json:"output"`
	Pretty  bool   `json:"pretty"`
	Debug   bool   `json:"debug"`
	Quiet   bool   `json:"quiet"`

	// MCP
	MCPReadOnly bool `json:"mcp_readonly"`

	// Auth token (runtime only, not persisted)
	RawToken string `json:"-"`
}

// DefaultConfigPath returns the full path to the default config file.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, DefaultConfigDir, DefaultConfigFile)
}

// DefaultTokenPath returns the full path to the JWT token cache.
func DefaultTokenPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, DefaultConfigDir, DefaultTokenFile)
}

// Load builds a Config by merging all sources in priority order.
// configPath is the path to the JSON config file (from flag or default).
// overrides contains values set by CLI flags (non-zero values win).
func Load(configPath string, overrides *Config) (*Config, error) {
	cfg := &Config{
		Timeout: DefaultTimeout,
		Output:  DefaultOutput,
		Pretty:  true,
	}

	// 4. Load from config file (lowest priority)
	if configPath == "" {
		configPath = DefaultConfigPath()
	}
	if err := loadFile(cfg, configPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading config file %s: %w", configPath, err)
	}

	// 3. Load from .env in current working directory
	loadDotEnv(cfg, ".env")

	// 2. Load from environment variables
	loadEnv(cfg)

	// 1. Apply CLI flag overrides (highest priority)
	if overrides != nil {
		applyOverrides(cfg, overrides)
	}

	// Set sensible defaults for output format
	if cfg.Output == "" {
		cfg.Output = DefaultOutput
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}

	return cfg, nil
}

// Validate checks that required fields are present.
func (c *Config) Validate() error {
	if c.RawToken != "" {
		// Token provided directly — no user/password needed
		if c.URL == "" {
			return fmt.Errorf("--url is required")
		}
		return nil
	}
	if c.URL == "" {
		return fmt.Errorf("wazuh URL is required (--url, WAZUH_URL, or config file)")
	}
	if c.User == "" {
		return fmt.Errorf("wazuh user is required (--user, WAZUH_USER, or config file)")
	}
	if c.Password == "" {
		return fmt.Errorf("wazuh password is required (--password, WAZUH_PASSWORD, or config file)")
	}
	return nil
}

// Save writes the config to the given path with 0600 permissions.
// Creates parent directories as needed.
func (c *Config) Save(path string) error {
	if path == "" {
		path = DefaultConfigPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}

// loadFile reads and merges a JSON config file into cfg.
func loadFile(cfg *Config, path string) error {
	// Check permissions before reading
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	mode := info.Mode().Perm()
	if mode&0077 != 0 {
		fmt.Fprintf(os.Stderr, "WARNING: config file %s has loose permissions (%o); should be 600\n", path, mode)
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var fileCfg Config
	if err := json.NewDecoder(f).Decode(&fileCfg); err != nil {
		return fmt.Errorf("parsing config JSON: %w", err)
	}
	mergeInto(cfg, &fileCfg)
	return nil
}

// loadDotEnv reads KEY=VALUE pairs from a .env file into cfg.
func loadDotEnv(cfg *Config, path string) {
	env, err := parseDotEnv(path)
	if err != nil {
		return // .env is optional
	}
	applyEnvMap(cfg, env)
}

// loadEnv reads WAZUH_* environment variables into cfg.
func loadEnv(cfg *Config) {
	applyEnvMap(cfg, envMap())
}

// envMap returns relevant environment variables as a map.
func envMap() map[string]string {
	keys := []string{
		"WAZUH_URL", "WAZUH_USER", "WAZUH_PASSWORD", "WAZUH_TOKEN",
		"WAZUH_INSECURE", "WAZUH_CA_CERT", "WAZUH_CLIENT_CERT", "WAZUH_CLIENT_KEY",
		"WAZUH_TIMEOUT", "WAZUH_OUTPUT", "WAZUH_PRETTY",
		"WAZUH_INDEXER_URL", "WAZUH_INDEXER_USER", "WAZUH_INDEXER_PASSWORD", "WAZUH_INDEXER_INDEX",
		"WAZUH_MCP_READONLY",
	}
	m := make(map[string]string, len(keys))
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			m[k] = v
		}
	}
	return m
}

// applyEnvMap maps env key→value pairs onto cfg fields.
func applyEnvMap(cfg *Config, env map[string]string) {
	if v := env["WAZUH_URL"]; v != "" {
		cfg.URL = v
	}
	if v := env["WAZUH_USER"]; v != "" {
		cfg.User = v
	}
	if v := env["WAZUH_PASSWORD"]; v != "" {
		cfg.Password = v
	}
	if v := env["WAZUH_TOKEN"]; v != "" {
		cfg.RawToken = v
	}
	if v := env["WAZUH_INSECURE"]; v == "true" || v == "1" || v == "yes" {
		cfg.Insecure = true
	}
	if v := env["WAZUH_CA_CERT"]; v != "" {
		cfg.CACert = v
	}
	if v := env["WAZUH_CLIENT_CERT"]; v != "" {
		cfg.ClientCert = v
	}
	if v := env["WAZUH_CLIENT_KEY"]; v != "" {
		cfg.ClientKey = v
	}
	if v := env["WAZUH_OUTPUT"]; v != "" {
		cfg.Output = v
	}
	if v := env["WAZUH_INDEXER_URL"]; v != "" {
		cfg.IndexerURL = v
	}
	if v := env["WAZUH_INDEXER_USER"]; v != "" {
		cfg.IndexerUser = v
	}
	if v := env["WAZUH_INDEXER_PASSWORD"]; v != "" {
		cfg.IndexerPassword = v
	}
	if v := env["WAZUH_INDEXER_INDEX"]; v != "" {
		cfg.IndexerIndex = v
	}
	if v := env["WAZUH_MCP_READONLY"]; v == "true" || v == "1" || v == "yes" {
		cfg.MCPReadOnly = true
	}
}

// applyOverrides copies non-zero/non-empty values from src onto dst.
func applyOverrides(dst, src *Config) {
	if src.URL != "" {
		dst.URL = src.URL
	}
	if src.User != "" {
		dst.User = src.User
	}
	if src.Password != "" {
		dst.Password = src.Password
	}
	if src.RawToken != "" {
		dst.RawToken = src.RawToken
	}
	if src.Insecure {
		dst.Insecure = true
	}
	if src.CACert != "" {
		dst.CACert = src.CACert
	}
	if src.ClientCert != "" {
		dst.ClientCert = src.ClientCert
	}
	if src.ClientKey != "" {
		dst.ClientKey = src.ClientKey
	}
	if src.Timeout != 0 {
		dst.Timeout = src.Timeout
	}
	if src.Output != "" {
		dst.Output = src.Output
	}
	if src.Pretty {
		dst.Pretty = true
	}
	if src.Debug {
		dst.Debug = true
	}
	if src.Quiet {
		dst.Quiet = true
	}
	if src.IndexerURL != "" {
		dst.IndexerURL = src.IndexerURL
	}
	if src.IndexerUser != "" {
		dst.IndexerUser = src.IndexerUser
	}
	if src.IndexerPassword != "" {
		dst.IndexerPassword = src.IndexerPassword
	}
	if src.IndexerIndex != "" {
		dst.IndexerIndex = src.IndexerIndex
	}
	if src.MCPReadOnly {
		dst.MCPReadOnly = true
	}
}

// mergeInto copies non-zero values from src onto dst (for file loading).
func mergeInto(dst, src *Config) {
	applyOverrides(dst, src)
}

// parseDotEnv reads a .env file and returns its contents as a map.
func parseDotEnv(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip blank lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Remove optional "export " prefix
		line = strings.TrimPrefix(line, "export ")
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		// Strip surrounding quotes
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		result[key] = val
	}
	return result, scanner.Err()
}
