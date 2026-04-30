package mcp

import (
	"context"
	"os"

	"github.com/ba0f3/wazuh-cli/internal/client"
	"github.com/ba0f3/wazuh-cli/internal/indexer"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ServerOpts struct {
	Client       *client.Client
	Indexer      *indexer.Client
	ReadOnly     bool
	Quiet        bool
	Version      string
	IndexerIndex string
}

type Server struct {
	mcp   *mcp.Server
	opts  ServerOpts
	audit *AuditLogger
}

func NewServer(opts ServerOpts) *Server {
	version := opts.Version
	if version == "" {
		version = "dev"
	}

	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "wazuh-mcp",
			Title:   "Wazuh MCP Server",
			Version: version,
		},
		nil,
	)

	s := &Server{
		mcp:  mcpServer,
		opts: opts,
	}

	if !opts.Quiet {
		s.audit = NewAuditLogger(os.Stderr)
	}

	s.registerAgentTools()
	s.registerAlertTools()
	s.registerVulnTools()
	s.registerResources()

	return s
}

func (s *Server) Run(ctx context.Context) error {
	return s.mcp.Run(ctx, &mcp.StdioTransport{})
}
