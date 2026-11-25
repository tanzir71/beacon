package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tanzir71/beacon/internal/sites/supervisor"
)

func TestLiveLogStreamHasBoundedClientCount(t *testing.T) {
	server := &Server{logClients: make(chan struct{}, 1)}
	server.logClients <- struct{}{}
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://beacon.test/_beacon/api/logs/app", nil)
	response := httptest.NewRecorder()
	server.streamCommandLogs(response, request, "app", supervisor.Snapshot{})
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("full live-log pool returned %d, want 503", response.Code)
	}
}
