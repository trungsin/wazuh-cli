package mcp

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestAuditLogger_Log(t *testing.T) {
	var buf bytes.Buffer
	logger := NewAuditLogger(&buf)

	logger.Log(AuditEntry{
		Tool:     "agent_list",
		Status:   "ok",
		Duration: "5ms",
	})

	var entry AuditEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse audit log: %v", err)
	}

	if entry.Tool != "agent_list" {
		t.Errorf("expected tool=agent_list, got %s", entry.Tool)
	}
	if entry.Status != "ok" {
		t.Errorf("expected status=ok, got %s", entry.Status)
	}
	if entry.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}
}

func TestAuditLogger_Track(t *testing.T) {
	var buf bytes.Buffer
	logger := NewAuditLogger(&buf)

	done := logger.Track("agent_get")
	done(nil)

	var entry AuditEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse audit log: %v", err)
	}

	if entry.Tool != "agent_get" {
		t.Errorf("expected tool=agent_get, got %s", entry.Tool)
	}
	if entry.Status != "ok" {
		t.Errorf("expected status=ok, got %s", entry.Status)
	}
}

func TestAuditLogger_TrackError(t *testing.T) {
	var buf bytes.Buffer
	logger := NewAuditLogger(&buf)

	done := logger.Track("agent_get")
	done(json.Unmarshal([]byte("invalid"), nil))

	var entry AuditEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse audit log: %v", err)
	}

	if entry.Status != "error" {
		t.Errorf("expected status=error, got %s", entry.Status)
	}
	if entry.Error == "" {
		t.Error("expected non-empty error field")
	}
}

func TestAuditLogger_NilSafe(t *testing.T) {
	var logger *AuditLogger
	done := logger.Track("test")
	done(nil) // should not panic
}

func TestTextResult(t *testing.T) {
	result := textResult(json.RawMessage(`{"id":"001"}`))
	if result.IsError {
		t.Error("expected IsError=false")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content, got %d", len(result.Content))
	}
}

func TestTextResult_Nil(t *testing.T) {
	result := textResult(nil)
	if result.IsError {
		t.Error("expected IsError=false")
	}
}

func TestTextResult_Null(t *testing.T) {
	result := textResult(json.RawMessage("null"))
	if result.IsError {
		t.Error("expected IsError=false")
	}
}

func TestErrorResult(t *testing.T) {
	result := errorResult(json.Unmarshal([]byte("bad"), nil))
	if !result.IsError {
		t.Error("expected IsError=true")
	}
}

func TestValidAgentID(t *testing.T) {
	valid := []string{"0", "001", "12345"}
	invalid := []string{"", "abc", "001/../../", "123456", "-1"}

	for _, id := range valid {
		if !validAgentID.MatchString(id) {
			t.Errorf("expected %q to be valid", id)
		}
	}
	for _, id := range invalid {
		if validAgentID.MatchString(id) {
			t.Errorf("expected %q to be invalid", id)
		}
	}
}
