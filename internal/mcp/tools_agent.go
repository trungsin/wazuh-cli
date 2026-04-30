package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/ba0f3/wazuh-cli/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var validAgentID = regexp.MustCompile(`^[0-9]{1,5}$`)

type AgentListInput struct {
	Status string `json:"status,omitempty" jsonschema:"description=Filter by status: active/disconnected/pending/never_connected"`
	Group  string `json:"group,omitempty" jsonschema:"description=Filter by group name"`
	Query  string `json:"query,omitempty" jsonschema:"description=WQL query filter"`
	Limit  int    `json:"limit,omitempty" jsonschema:"description=Max results (default 500)"`
	Offset int    `json:"offset,omitempty" jsonschema:"description=Pagination offset"`
}

type AgentGetInput struct {
	AgentID string `json:"agent_id" jsonschema:"description=Agent ID,required"`
}

type AgentSummaryInput struct{}

func (s *Server) registerAgentTools() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "agent_list",
		Description: "List and search Wazuh agents. Returns agent details including ID, name, IP, OS, status, and group membership.",
	}, s.handleAgentList)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "agent_get",
		Description: "Get detailed information for a specific Wazuh agent by ID.",
	}, s.handleAgentGet)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "agent_summary",
		Description: "Get agent status summary showing counts of active, disconnected, pending, and never-connected agents.",
	}, s.handleAgentSummary)
}

func (s *Server) handleAgentList(_ context.Context, _ *mcp.CallToolRequest, input AgentListInput) (*mcp.CallToolResult, any, error) {
	var toolErr error
	done := s.audit.Track("agent_list")
	defer func() { done(toolErr) }()

	limit := input.Limit
	if limit == 0 {
		limit = 500
	}

	params := &client.QueryParams{
		Limit:  limit,
		Offset: input.Offset,
		Query:  input.Query,
	}
	if input.Status != "" || input.Group != "" {
		params.Filters = make(map[string]string)
		if input.Status != "" {
			params.Filters["status"] = input.Status
		}
		if input.Group != "" {
			params.Filters["group"] = input.Group
		}
	}

	resp, err := s.opts.Client.Get("/agents", params)
	if err != nil {
		toolErr = err
		return errorResult(err), nil, nil
	}

	return textResult(resp.Items()), nil, nil
}

func (s *Server) handleAgentGet(_ context.Context, _ *mcp.CallToolRequest, input AgentGetInput) (*mcp.CallToolResult, any, error) {
	var toolErr error
	done := s.audit.Track("agent_get")
	defer func() { done(toolErr) }()

	if !validAgentID.MatchString(input.AgentID) {
		toolErr = fmt.Errorf("invalid agent_id: must be numeric (e.g. 001)")
		return errorResult(toolErr), nil, nil
	}

	resp, err := s.opts.Client.Get(fmt.Sprintf("/agents/%s", input.AgentID), nil)
	if err != nil {
		toolErr = err
		return errorResult(err), nil, nil
	}

	return textResult(resp.Items()), nil, nil
}

func (s *Server) handleAgentSummary(_ context.Context, _ *mcp.CallToolRequest, _ AgentSummaryInput) (*mcp.CallToolResult, any, error) {
	var toolErr error
	done := s.audit.Track("agent_summary")
	defer func() { done(toolErr) }()

	resp, err := s.opts.Client.Get("/agents/summary/status", nil)
	if err != nil {
		toolErr = err
		return errorResult(err), nil, nil
	}

	return textResult(resp.Data), nil, nil
}

func textResult(data json.RawMessage) *mcp.CallToolResult {
	text := string(data)
	if text == "" || text == "null" {
		text = "[]"
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

func errorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: err.Error()},
		},
		IsError: true,
	}
}
