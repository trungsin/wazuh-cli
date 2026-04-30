package mcp

import (
	"context"
	"fmt"

	"github.com/ba0f3/wazuh-cli/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type VulnListInput struct {
	AgentID  string `json:"agent_id" jsonschema:"description=Agent ID to query,required"`
	Severity string `json:"severity,omitempty" jsonschema:"description=Filter by severity: critical/high/medium/low"`
	Limit    int    `json:"limit,omitempty" jsonschema:"description=Max results (default 500)"`
	Offset   int    `json:"offset,omitempty" jsonschema:"description=Pagination offset"`
}

type VulnSummaryInput struct {
	AgentID string `json:"agent_id" jsonschema:"description=Agent ID to query,required"`
}

func (s *Server) registerVulnTools() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "vulnerability_list",
		Description: "List CVE vulnerabilities detected on a specific Wazuh agent. Filter by severity level.",
	}, s.handleVulnList)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "vulnerability_summary",
		Description: "Get vulnerability severity breakdown (critical/high/medium/low counts) for a specific agent.",
	}, s.handleVulnSummary)
}

func (s *Server) handleVulnList(_ context.Context, _ *mcp.CallToolRequest, input VulnListInput) (*mcp.CallToolResult, any, error) {
	var toolErr error
	done := s.audit.Track("vulnerability_list")
	defer func() { done(toolErr) }()

	if !validAgentID.MatchString(input.AgentID) {
		toolErr = fmt.Errorf("invalid agent_id: must be numeric (e.g. 001)")
		return errorResult(toolErr), nil, nil
	}

	limit := input.Limit
	if limit == 0 {
		limit = 500
	}
	if limit > 500 {
		limit = 500
	}

	params := &client.QueryParams{
		Limit:  limit,
		Offset: input.Offset,
	}
	if input.Severity != "" {
		params.Filters = map[string]string{"severity": input.Severity}
	}

	resp, err := s.opts.Client.Get(fmt.Sprintf("/vulnerability/%s", input.AgentID), params)
	if err != nil {
		toolErr = err
		return errorResult(err), nil, nil
	}

	return textResult(resp.Items()), nil, nil
}

func (s *Server) handleVulnSummary(_ context.Context, _ *mcp.CallToolRequest, input VulnSummaryInput) (*mcp.CallToolResult, any, error) {
	var toolErr error
	done := s.audit.Track("vulnerability_summary")
	defer func() { done(toolErr) }()

	if !validAgentID.MatchString(input.AgentID) {
		toolErr = fmt.Errorf("invalid agent_id: must be numeric (e.g. 001)")
		return errorResult(toolErr), nil, nil
	}

	resp, err := s.opts.Client.Get(fmt.Sprintf("/vulnerability/%s/summary/severity", input.AgentID), nil)
	if err != nil {
		toolErr = err
		return errorResult(err), nil, nil
	}

	return textResult(resp.Data), nil, nil
}
