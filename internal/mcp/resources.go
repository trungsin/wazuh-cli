package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerResources() {
	s.mcp.AddResourceTemplate(
		&mcp.ResourceTemplate{
			Name:        "Wazuh Agent Profile",
			URITemplate: "wazuh://agent/{id}",
			Description: "Full agent profile including name, IP, OS, status, and group membership",
			MIMEType:    "application/json",
		},
		s.handleAgentResource,
	)

	if s.opts.Indexer != nil {
		s.mcp.AddResourceTemplate(
			&mcp.ResourceTemplate{
				Name:        "Wazuh Alert",
				URITemplate: "wazuh://alert/{id}",
				Description: "Alert document from OpenSearch indexer",
				MIMEType:    "application/json",
			},
			s.handleAlertResource,
		)
	}
}

func (s *Server) handleAgentResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	id := extractURIParam(req.Params.URI, "wazuh://agent/")
	if id == "" || !validAgentID.MatchString(id) {
		return nil, fmt.Errorf("invalid agent ID in URI")
	}

	resp, err := s.opts.Client.Get(fmt.Sprintf("/agents/%s", id), nil)
	if err != nil {
		return nil, fmt.Errorf("agent lookup failed: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(resp.Items()),
			},
		},
	}, nil
}

func (s *Server) handleAlertResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	id := extractURIParam(req.Params.URI, "wazuh://alert/")
	if id == "" || strings.Contains(id, "..") || strings.ContainsAny(id, "/\\") {
		return nil, fmt.Errorf("invalid alert ID in URI")
	}

	hit, err := s.opts.Indexer.Get(s.indexPattern(), id)
	if err != nil {
		return nil, fmt.Errorf("alert lookup failed: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(hit.Source),
			},
		},
	}, nil
}

func extractURIParam(uri, prefix string) string {
	if !strings.HasPrefix(uri, prefix) {
		return ""
	}
	return strings.TrimPrefix(uri, prefix)
}
