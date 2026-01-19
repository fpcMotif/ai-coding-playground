package logger

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestLoggerCreation(t *testing.T) {
	log := New()
	if log == nil {
		t.Error("New() returned nil")
	}
	if log.logger == nil {
		t.Error("logger is nil")
	}
	if log.handler == nil {
		t.Error("handler is nil")
	}
}

func TestLoggerStructuredOutput(t *testing.T) {
	// Capture stderr (slog outputs to stdout by default)
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	log := New()
	log.Info("test message", "key", "value", "number", 42)

	w.Close()
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := strings.TrimSpace(buf.String())

	// Verify JSON output
	var data map[string]interface{}
	err := json.Unmarshal([]byte(output), &data)
	if err != nil {
		t.Fatalf("Output is not valid JSON: %v (output: %s)", err, output)
	}

	// Check for expected fields
	if msg, ok := data["msg"]; !ok || msg != "test message" {
		t.Errorf("Expected msg field with value 'test message', got %v", msg)
	}

	if key, ok := data["key"]; !ok || key != "value" {
		t.Errorf("Expected key field with value 'value', got %v", key)
	}

	if number, ok := data["number"]; !ok || number != float64(42) {
		t.Errorf("Expected number field with value 42, got %v", number)
	}
}

func TestLoggerLevels(t *testing.T) {
	log := New()

	tests := []struct {
		name string
		fn   func(msg string, args ...any)
	}{
		{"Info", log.Info},
		{"Error", log.Error},
		{"Warn", log.Warn},
		{"Debug", log.Debug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that methods don't panic - actual output verification
			// is done in other tests
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s panicked: %v", tt.name, r)
				}
			}()

			tt.fn("test", "key", "value")
		})
	}
}

func TestLoggerWith(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	log := New()
	ctxLog := log.With("request_id", "12345", "user", "alice")
	ctxLog.Info("user action", "action", "login")

	w.Close()
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := strings.TrimSpace(buf.String())

	// Verify context fields are present
	var data map[string]interface{}
	err := json.Unmarshal([]byte(output), &data)
	if err != nil {
		t.Fatalf("Output is not valid JSON: %v (output: %s)", err, output)
	}

	if id, ok := data["request_id"]; !ok || id != "12345" {
		t.Errorf("Expected request_id field, got %v", id)
	}

	if user, ok := data["user"]; !ok || user != "alice" {
		t.Errorf("Expected user field, got %v", user)
	}

	if action, ok := data["action"]; !ok || action != "login" {
		t.Errorf("Expected action field, got %v", action)
	}
}

func TestLoggerWithGroup(t *testing.T) {
	log := New()
	grpLog := log.WithGroup("component")
	if grpLog == nil {
		t.Error("WithGroup() returned nil")
	}
}

func TestLoggerDeprecatedMethods(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	log := New()
	log.Infof("deprecated %s method", "info")

	w.Close()
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := strings.TrimSpace(buf.String())

	// Verify output is still valid JSON
	var data map[string]interface{}
	err := json.Unmarshal([]byte(output), &data)
	if err != nil {
		t.Fatalf("Output is not valid JSON: %v (output: %s)", err, output)
	}

	// Check message is there
	if msg, ok := data["msg"]; !ok || msg == "" {
		t.Errorf("Expected msg field, got %v", msg)
	}
}

func TestLoggerFatalCalls(t *testing.T) {
	// We can't easily test Fatal() since it calls os.Exit
	// Just verify the method exists and doesn't panic on the wrapper

	log := New()

	// Test that With returns a logger
	ctxLog := log.With("test", "value")
	if ctxLog == nil {
		t.Error("With() returned nil")
	}

	// Test that WithGroup returns a logger
	grpLog := log.WithGroup("test_group")
	if grpLog == nil {
		t.Error("WithGroup() returned nil")
	}
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name string
		args []any
		want int
	}{
		{"empty", []any{}, 0},
		{"single pair", []any{"key", "value"}, 1},
		{"multiple pairs", []any{"k1", "v1", "k2", "v2", "k3", "v3"}, 3},
		{"odd number", []any{"k1", "v1", "k2"}, 1}, // Last unpaired should be skipped
		{"non-string key", []any{123, "value"}, 0},  // Non-string keys are skipped
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := parseArgs(tt.args...)
			if len(attrs) != tt.want {
				t.Errorf("parseArgs() returned %d attrs, want %d", len(attrs), tt.want)
			}
		})
	}
}

func TestLoggerOutputFormat(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	log := New()
	log.Info("test event", "status", "success", "code", 200)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := strings.TrimSpace(buf.String())

	// Should be valid JSON
	var data map[string]interface{}
	err := json.Unmarshal([]byte(output), &data)
	if err != nil {
		t.Fatalf("Output is not valid JSON: %s", output)
	}

	// Should have level and message
	if _, ok := data["level"]; !ok {
		t.Error("Missing 'level' field")
	}
	if _, ok := data["msg"]; !ok {
		t.Error("Missing 'msg' field")
	}

	// Should have time
	if _, ok := data["time"]; !ok {
		t.Error("Missing 'time' field")
	}
}
