package webserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebServerRelayLogging(t *testing.T) {
	os.Setenv("AUTH_KEY", "test-auth-key")
	defer os.Unsetenv("AUTH_KEY")

	server, err := New(18080)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	tmpDir := t.TempDir()
	relayFilePath := filepath.Join(tmpDir, "relays.txt")
	os.WriteFile(relayFilePath, []byte("socks5://127.0.0.1:1080\n"), 0644)

	os.Setenv("RELAY_POOL_URL", relayFilePath)
	os.Setenv("RELAY_SUCCESS_COUNT", "1")
	defer func() {
		os.Unsetenv("RELAY_POOL_URL")
		os.Unsetenv("RELAY_SUCCESS_COUNT")
	}()

	dummyYAML := `proxies:
  - name: "node-1"
    type: socks5
    server: 127.0.0.1
    port: 1080
`
	req := httptest.NewRequest(http.MethodPost, "/speedtest", bytes.NewBufferString(dummyYAML))
	req.Header.Set("Authorization", "Bearer test-auth-key")
	rec := httptest.NewRecorder()

	server.handleSpeedTest(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWebServerDirectLogging(t *testing.T) {
	os.Setenv("AUTH_KEY", "test-auth-key")
	defer os.Unsetenv("AUTH_KEY")

	server, err := New(18080)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	os.Unsetenv("RELAY_POOL_URL")

	dummyYAML := `proxies:
  - name: "node-direct"
    type: socks5
    server: 127.0.0.1
    port: 1080
`
	req := httptest.NewRequest(http.MethodPost, "/speedtest", bytes.NewBufferString(dummyYAML))
	req.Header.Set("Authorization", "Bearer test-auth-key")
	rec := httptest.NewRecorder()

	server.handleSpeedTest(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWebServerHealth(t *testing.T) {
	os.Setenv("AUTH_KEY", "test-auth-key")
	defer os.Unsetenv("AUTH_KEY")

	server, err := New(18080)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Errorf("expected status ok body, got %s", rec.Body.String())
	}
}
