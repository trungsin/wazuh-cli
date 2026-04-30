package mcp

import (
	"encoding/json"
	"io"
	"sync"
	"time"
)

type AuditEntry struct {
	Timestamp string `json:"timestamp"`
	Tool      string `json:"tool"`
	Status    string `json:"status"`
	Duration  string `json:"duration"`
	Error     string `json:"error,omitempty"`
}

type AuditLogger struct {
	writer io.Writer
	mu     sync.Mutex
}

func NewAuditLogger(w io.Writer) *AuditLogger {
	return &AuditLogger{writer: w}
}

func (a *AuditLogger) Track(tool string) func(err error) {
	if a == nil {
		return func(error) {}
	}
	start := time.Now()
	return func(err error) {
		status := "ok"
		errMsg := ""
		if err != nil {
			status = "error"
			errMsg = err.Error()
		}
		a.Log(AuditEntry{
			Tool:     tool,
			Status:   status,
			Duration: time.Since(start).String(),
			Error:    errMsg,
		})
	}
}

func (a *AuditLogger) Log(entry AuditEntry) {
	if a == nil {
		return
	}
	entry.Timestamp = time.Now().UTC().Format(time.RFC3339)
	a.mu.Lock()
	defer a.mu.Unlock()
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	data = append(data, '\n')
	a.writer.Write(data) //nolint:errcheck
}
