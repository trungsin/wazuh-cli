package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type AlertListInput struct {
	AgentID   string `json:"agent_id,omitempty" jsonschema:"description=Filter by agent ID"`
	AgentName string `json:"agent_name,omitempty" jsonschema:"description=Filter by agent name"`
	RuleID    string `json:"rule_id,omitempty" jsonschema:"description=Filter by rule ID"`
	RuleGroup string `json:"rule_group,omitempty" jsonschema:"description=Filter by rule group"`
	Level     int    `json:"level,omitempty" jsonschema:"description=Minimum rule level (1-15)"`
	Query     string `json:"query,omitempty" jsonschema:"description=Raw Lucene query string"`
	From      string `json:"from,omitempty" jsonschema:"description=Start time (default: now-24h). Examples: now-1h or 2024-01-01T00:00:00Z"`
	To        string `json:"to,omitempty" jsonschema:"description=End time (default: now)"`
	Limit     int    `json:"limit,omitempty" jsonschema:"description=Max results (default 50)"`
}

type AlertGetInput struct {
	DocID string `json:"doc_id" jsonschema:"description=OpenSearch document ID,required"`
	Index string `json:"index,omitempty" jsonschema:"description=Index pattern override (default: wazuh-alerts-4.x-*)"`
}

type AlertStatsInput struct {
	GroupBy string `json:"group_by,omitempty" jsonschema:"description=Group by field: level/agent/rule (default: level)"`
	Level   int    `json:"level,omitempty" jsonschema:"description=Minimum rule level"`
	AgentID string `json:"agent_id,omitempty" jsonschema:"description=Filter by agent ID"`
	From    string `json:"from,omitempty" jsonschema:"description=Start time (default: now-24h)"`
	To      string `json:"to,omitempty" jsonschema:"description=End time (default: now)"`
}

func (s *Server) registerAlertTools() {
	if s.opts.Indexer == nil {
		return
	}

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "alert_list",
		Description: "Query recent Wazuh alerts from the OpenSearch indexer. Filter by agent, rule, severity level, and time range.",
	}, s.handleAlertList)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "alert_get",
		Description: "Get a specific Wazuh alert by its OpenSearch document ID.",
	}, s.handleAlertGet)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "alert_stats",
		Description: "Get aggregated alert statistics grouped by level, agent, or rule.",
	}, s.handleAlertStats)
}

func (s *Server) indexPattern() string {
	if s.opts.IndexerIndex != "" {
		return s.opts.IndexerIndex
	}
	return "wazuh-alerts-4.x-*"
}

func (s *Server) handleAlertList(_ context.Context, _ *mcp.CallToolRequest, input AlertListInput) (*mcp.CallToolResult, any, error) {
	var toolErr error
	done := s.audit.Track("alert_list")
	defer func() { done(toolErr) }()

	from := input.From
	if from == "" {
		from = "now-24h"
	}
	to := input.To
	if to == "" {
		to = "now"
	}
	limit := input.Limit
	if limit == 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	query := buildAlertSearchQuery(limit, input.Level, input.AgentID, input.AgentName, input.RuleID, input.RuleGroup, input.Query, from, to)

	resp, err := s.opts.Indexer.Search(s.indexPattern(), query)
	if err != nil {
		toolErr = err
		return errorResult(err), nil, nil
	}

	var docs []json.RawMessage
	for _, hit := range resp.Hits.Hits {
		docs = append(docs, hit.Source)
	}

	if len(docs) == 0 {
		return textResult(json.RawMessage("[]")), nil, nil
	}

	data, err := json.Marshal(docs)
	if err != nil {
		toolErr = fmt.Errorf("encoding results: %w", err)
		return errorResult(toolErr), nil, nil
	}

	return textResult(json.RawMessage(data)), nil, nil
}

func (s *Server) handleAlertGet(_ context.Context, _ *mcp.CallToolRequest, input AlertGetInput) (*mcp.CallToolResult, any, error) {
	var toolErr error
	done := s.audit.Track("alert_get")
	defer func() { done(toolErr) }()

	if input.DocID == "" {
		toolErr = fmt.Errorf("doc_id is required")
		return errorResult(toolErr), nil, nil
	}
	if strings.Contains(input.DocID, "..") || strings.ContainsAny(input.DocID, "/\\") {
		toolErr = fmt.Errorf("invalid doc_id")
		return errorResult(toolErr), nil, nil
	}

	index := input.Index
	if index == "" {
		index = s.indexPattern()
	}

	hit, err := s.opts.Indexer.Get(index, input.DocID)
	if err != nil {
		toolErr = err
		return errorResult(err), nil, nil
	}

	return textResult(hit.Source), nil, nil
}

func (s *Server) handleAlertStats(_ context.Context, _ *mcp.CallToolRequest, input AlertStatsInput) (*mcp.CallToolResult, any, error) {
	var toolErr error
	done := s.audit.Track("alert_stats")
	defer func() { done(toolErr) }()

	from := input.From
	if from == "" {
		from = "now-24h"
	}
	to := input.To
	if to == "" {
		to = "now"
	}
	groupBy := input.GroupBy
	if groupBy == "" {
		groupBy = "level"
	}

	query := buildAlertStatsQuery(groupBy, input.Level, input.AgentID, from, to)

	resp, err := s.opts.Indexer.Search(s.indexPattern(), query)
	if err != nil {
		toolErr = err
		return errorResult(err), nil, nil
	}

	if agg, ok := resp.Aggregations["stats"]; ok {
		data, err := json.Marshal(agg.Buckets)
		if err != nil {
			return errorResult(fmt.Errorf("encoding stats: %w", err)), nil, nil
		}
		return textResult(json.RawMessage(data)), nil, nil
	}

	return textResult(json.RawMessage("[]")), nil, nil
}

func buildAlertSearchQuery(limit, level int, agentID, agentName, ruleID, ruleGroup, queryStr, from, to string) map[string]interface{} {
	must := []map[string]interface{}{}

	if level > 0 {
		must = append(must, map[string]interface{}{
			"range": map[string]interface{}{
				"rule.level": map[string]interface{}{"gte": level},
			},
		})
	}
	if agentID != "" {
		must = append(must, map[string]interface{}{
			"term": map[string]interface{}{"agent.id": agentID},
		})
	}
	if agentName != "" {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{"agent.name": agentName},
		})
	}
	if ruleID != "" {
		must = append(must, map[string]interface{}{
			"term": map[string]interface{}{"rule.id": ruleID},
		})
	}
	if ruleGroup != "" {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{"rule.groups": ruleGroup},
		})
	}
	if queryStr != "" {
		must = append(must, map[string]interface{}{
			"query_string": map[string]interface{}{"query": queryStr},
		})
	}

	must = append(must, map[string]interface{}{
		"range": map[string]interface{}{
			"timestamp": map[string]interface{}{
				"gte": from,
				"lte": to,
			},
		},
	})

	return map[string]interface{}{
		"size": limit,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": must,
			},
		},
		"sort": []map[string]interface{}{
			{"timestamp": "desc"},
		},
	}
}

func buildAlertStatsQuery(groupBy string, level int, agentID, from, to string) map[string]interface{} {
	must := []map[string]interface{}{}

	if level > 0 {
		must = append(must, map[string]interface{}{
			"range": map[string]interface{}{
				"rule.level": map[string]interface{}{"gte": level},
			},
		})
	}
	if agentID != "" {
		must = append(must, map[string]interface{}{
			"term": map[string]interface{}{"agent.id": agentID},
		})
	}

	must = append(must, map[string]interface{}{
		"range": map[string]interface{}{
			"timestamp": map[string]interface{}{
				"gte": from,
				"lte": to,
			},
		},
	})

	fieldMap := map[string]string{
		"level": "rule.level",
		"agent": "agent.name",
		"rule":  "rule.id",
	}

	aggField, ok := fieldMap[groupBy]
	if !ok {
		aggField = "rule.level"
	}

	return map[string]interface{}{
		"size": 0,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": must,
			},
		},
		"aggs": map[string]interface{}{
			"stats": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": aggField,
					"size":  50,
				},
			},
		},
	}
}
