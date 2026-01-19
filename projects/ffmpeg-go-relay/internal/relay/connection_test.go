package relay

import (
	"testing"
	"time"
)

func TestActiveConnectionTracking(t *testing.T) {
	clearActiveConnections()
	t.Cleanup(clearActiveConnections)

	requestID := "req-test-1"
	info := ConnectionInfo{
		RequestID:  requestID,
		ClientAddr: "1.2.3.4:5555",
		Upstream:   "rtmp://example.com/app",
		StartTime:  time.Now(),
		State:      "connecting",
	}

	trackConnectionStart(info)

	if got := GetActiveConnectionCount(); got != 1 {
		t.Fatalf("active connections = %d, want 1", got)
	}

	updateConnectionState(requestID, "relaying")

	connections := GetActiveConnectionsList()
	found := false
	for _, conn := range connections {
		if conn.RequestID == requestID {
			found = true
			if conn.State != "relaying" {
				t.Fatalf("state = %s, want relaying", conn.State)
			}
		}
	}
	if !found {
		t.Fatalf("connection %s not found", requestID)
	}

	trackConnectionEnd(requestID)
	if got := GetActiveConnectionCount(); got != 0 {
		t.Fatalf("active connections after delete = %d, want 0", got)
	}
}

func clearActiveConnections() {
	activeConnections.Range(func(key, value any) bool {
		activeConnections.Delete(key)
		return true
	})
}
