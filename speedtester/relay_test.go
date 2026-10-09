package speedtester

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/metacubex/mihomo/adapter"
)

func TestParseProxyFromURL(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		expectError  bool
		expectedTyp  string
		expectedTLS  bool
		expectedUser string
		expectedPwd  string
	}{
		{
			name:        "standard socks5",
			url:         "socks5://101.5.21.225:7894",
			expectError: false,
			expectedTyp: "socks5",
		},
		{
			name:         "socks5 with auth",
			url:          "socks5://alice:secret123@1.2.3.4:1080",
			expectError:  false,
			expectedTyp:  "socks5",
			expectedUser: "alice",
			expectedPwd:  "secret123",
		},
		{
			name:        "http proxy",
			url:         "http://12.34.56.78:8080",
			expectError: false,
			expectedTyp: "http",
		},
		{
			name:        "https proxy",
			url:         "https://98.76.54.32:8443",
			expectError: false,
			expectedTyp: "http",
			expectedTLS: true,
		},
		{
			name:        "unsupported scheme socks4",
			url:         "socks4://1.1.1.1:1080",
			expectError: true,
		},
		{
			name:        "invalid missing port",
			url:         "socks5://1.1.1.1",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := parseProxyFromURL(tc.url, 0)
			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error for %s, got nil", tc.url)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tc.url, err)
			}

			if cfg["type"] != tc.expectedTyp {
				t.Errorf("expected type %s, got %v", tc.expectedTyp, cfg["type"])
			}
			if tc.expectedTLS && cfg["tls"] != true {
				t.Errorf("expected tls=true, got %v", cfg["tls"])
			}
			if tc.expectedUser != "" && cfg["username"] != tc.expectedUser {
				t.Errorf("expected username %s, got %v", tc.expectedUser, cfg["username"])
			}
			if tc.expectedPwd != "" && cfg["password"] != tc.expectedPwd {
				t.Errorf("expected password %s, got %v", tc.expectedPwd, cfg["password"])
			}
		})
	}
}

func TestParseRelayPoolContent(t *testing.T) {
	t.Run("parse clash yaml format", func(t *testing.T) {
		yamlData := []byte(`
proxies:
  - name: "cn-relay-1"
    type: http
    server: 127.0.0.1
    port: 8080
  - name: "cn-relay-2"
    type: socks5
    server: 127.0.0.1
    port: 1080
`)
		proxies := parseRelayPoolContent(yamlData)
		if len(proxies) != 2 {
			t.Fatalf("expected 2 proxies, got %d", len(proxies))
		}
		if proxies[0].Name() != "cn-relay-1" {
			t.Errorf("expected name cn-relay-1, got %s", proxies[0].Name())
		}
	})

	t.Run("parse line-by-line urls format", func(t *testing.T) {
		lines := []byte(`# comment line
socks5://127.0.0.1:1080
http://127.0.0.1:8080

// another comment
socks4://127.0.0.1:1080
socks5://user:pass@127.0.0.1:1081
`)
		proxies := parseRelayPoolContent(lines)
		// socks4 should be ignored, 3 proxies should be parsed
		if len(proxies) != 3 {
			t.Fatalf("expected 3 valid proxies, got %d", len(proxies))
		}
	})
}

func TestLoadRelayPoolFileAndCache(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "relays.txt")
	content := "socks5://127.0.0.1:1080\nhttp://127.0.0.1:8080\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	proxies, err := LoadRelayPool(filePath)
	if err != nil {
		t.Fatalf("LoadRelayPool failed: %v", err)
	}
	if len(proxies) != 2 {
		t.Fatalf("expected 2 proxies, got %d", len(proxies))
	}

	// Test cache hit
	cachedProxies, err := LoadRelayPool(filePath)
	if err != nil {
		t.Fatalf("LoadRelayPool from cache failed: %v", err)
	}
	if len(cachedProxies) != 2 {
		t.Fatalf("expected 2 cached proxies, got %d", len(cachedProxies))
	}

	// Test empty URL
	emptyProxies, err := LoadRelayPool("")
	if err != nil || emptyProxies != nil {
		t.Fatalf("expected nil, nil for empty url, got %v, %v", emptyProxies, err)
	}
}

func TestSpeedTesterRelayEnvironmentVariables(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "env_relays.txt")
	content := "socks5://127.0.0.1:1080\nhttp://127.0.0.1:8080\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Unset env vars
	os.Unsetenv("RELAY_POOL_URL")
	os.Unsetenv("RELAY_SUCCESS_COUNT")
	os.Unsetenv("RELAY_COUNT")

	stDirect := New(&Config{
		ServerURL: "https://speed.cloudflare.com",
	})
	if stDirect.RelayProxyCount() != 0 {
		t.Errorf("expected 0 relay proxies when env unset, got %d", stDirect.RelayProxyCount())
	}

	// Set env vars
	os.Setenv("RELAY_POOL_URL", filePath)
	os.Setenv("RELAY_SUCCESS_COUNT", "2")
	defer func() {
		os.Unsetenv("RELAY_POOL_URL")
		os.Unsetenv("RELAY_SUCCESS_COUNT")
	}()

	stRelay := New(&Config{
		ServerURL: "https://speed.cloudflare.com",
	})
	if stRelay.RelayProxyCount() != 2 {
		t.Errorf("expected 2 relay proxies from env, got %d", stRelay.RelayProxyCount())
	}
	if stRelay.config.RelaySuccessCount != 2 {
		t.Errorf("expected RelaySuccessCount 2, got %d", stRelay.config.RelaySuccessCount)
	}
}

func startMockHTTPTunnel(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect {
			destConn, err := net.DialTimeout("tcp", r.Host, 2*time.Second)
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "hijack not supported", http.StatusInternalServerError)
				destConn.Close()
				return
			}
			clientConn, _, err := hijacker.Hijack()
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				destConn.Close()
				return
			}
			clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
			go func() {
				defer destConn.Close()
				defer clientConn.Close()
				var buf [4096]byte
				for {
					n, err := clientConn.Read(buf[:])
					if n > 0 {
						destConn.Write(buf[:n])
					}
					if err != nil {
						break
					}
				}
			}()
			go func() {
				defer destConn.Close()
				defer clientConn.Close()
				var buf [4096]byte
				for {
					n, err := destConn.Read(buf[:])
					if n > 0 {
						clientConn.Write(buf[:n])
					}
					if err != nil {
						break
					}
				}
			}()
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
}

func TestRelayPoolThresholdEvaluation(t *testing.T) {
	// Destination server that will be reached through the chain
	destServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("destination-ok"))
	}))
	defer destServer.Close()

	// Target node is an HTTP tunnel proxy
	targetProxyServer := startMockHTTPTunnel(t)
	defer targetProxyServer.Close()

	_, targetPortStr, err := net.SplitHostPort(targetProxyServer.Listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to split host port: %v", err)
	}
	var targetPort int
	fmt.Sscanf(targetPortStr, "%d", &targetPort)

	targetCfg := map[string]any{
		"name":   "target-node",
		"type":   "http",
		"server": "127.0.0.1",
		"port":   targetPort,
	}
	targetProxy, err := adapter.ParseProxy(targetCfg)
	if err != nil {
		t.Fatalf("failed to parse target proxy: %v", err)
	}
	cTarget := &CProxy{Proxy: targetProxy, Config: targetCfg}

	// Relay 1 is a working HTTP tunnel proxy
	relay1Server := startMockHTTPTunnel(t)
	defer relay1Server.Close()

	_, relay1PortStr, _ := net.SplitHostPort(relay1Server.Listener.Addr().String())
	var relay1Port int
	fmt.Sscanf(relay1PortStr, "%d", &relay1Port)

	relay1Cfg := map[string]any{
		"name":   "relay-1",
		"type":   "http",
		"server": "127.0.0.1",
		"port":   relay1Port,
	}
	relay1Proxy, err := adapter.ParseProxy(relay1Cfg)
	if err != nil {
		t.Fatalf("failed to parse relay1: %v", err)
	}

	// Relay 2 is dead / points to an unused port
	relay2Cfg := map[string]any{
		"name":   "relay-2-dead",
		"type":   "http",
		"server": "127.0.0.1",
		"port":   59999,
	}
	relay2Proxy, err := adapter.ParseProxy(relay2Cfg)
	if err != nil {
		t.Fatalf("failed to parse relay2: %v", err)
	}

	st := &SpeedTester{
		config: &Config{
			ServerURL:         destServer.URL,
			Timeout:           1 * time.Second,
			RelaySuccessCount: 1,
			FastMode:          true,
		},
		relayProxies: []*CProxy{
			{Proxy: relay2Proxy, Config: relay2Cfg},
			{Proxy: relay2Proxy, Config: relay2Cfg},
		},
	}

	// Case 1: All relays dead -> must fail
	resFail := st.testProxyWithRelayPool("target-node", cTarget)
	if resFail.Latency > 0 {
		t.Errorf("expected failure (latency=0) when all relays are dead, got %v", resFail.Latency)
	}

	// Case 2: One working relay with required success = 1 -> should pass
	st.relayProxies = []*CProxy{
		{Proxy: relay1Proxy, Config: relay1Cfg},
		{Proxy: relay2Proxy, Config: relay2Cfg},
	}
	st.config.RelaySuccessCount = 1
	resPass := st.testProxyWithRelayPool("target-node", cTarget)
	if resPass.Latency == 0 {
		t.Errorf("expected success (latency > 0) when 1 relay succeeds out of required 1, got 0")
	}

	// Case 3: One working relay with required success = 2 -> must fail because only 1 succeeded
	st.config.RelaySuccessCount = 2
	resUnderThreshold := st.testProxyWithRelayPool("target-node", cTarget)
	if resUnderThreshold.Latency > 0 {
		t.Errorf("expected failure when success count (1) < required (2), got latency %v", resUnderThreshold.Latency)
	}
}

func TestSelectRelaySample(t *testing.T) {
	// Empty slice
	if selectRelaySample(nil, 5) != nil {
		t.Errorf("expected nil for empty input")
	}

	// Create 20 mock proxies
	total := 20
	proxies := make([]*CProxy, total)
	for i := 0; i < total; i++ {
		cfg := map[string]any{
			"name":   fmt.Sprintf("mock-relay-%d", i),
			"type":   "http",
			"server": "127.0.0.1",
			"port":   8000 + i,
		}
		p, _ := adapter.ParseProxy(cfg)
		proxies[i] = &CProxy{Proxy: p, Config: cfg}
	}

	// Case 1: sampleCount <= 0 -> returns all 20
	resAll := selectRelaySample(proxies, 0)
	if len(resAll) != total {
		t.Errorf("expected %d relays when sampleCount <= 0, got %d", total, len(resAll))
	}

	// Case 2: sampleCount >= total -> returns all 20
	resMore := selectRelaySample(proxies, 30)
	if len(resMore) != total {
		t.Errorf("expected %d relays when sampleCount >= total, got %d", total, len(resMore))
	}

	// Case 3: sampleCount < total (5 out of 20) -> returns exactly 5 distinct items
	sampleCount := 5
	resSample := selectRelaySample(proxies, sampleCount)
	if len(resSample) != sampleCount {
		t.Fatalf("expected %d relays, got %d", sampleCount, len(resSample))
	}
	seen := make(map[string]bool)
	for _, p := range resSample {
		if seen[p.Name()] {
			t.Errorf("duplicate relay found in sample: %s", p.Name())
		}
		seen[p.Name()] = true
	}
}

func TestRelaySampleCountEnvironmentVariables(t *testing.T) {
	// Clean env
	os.Unsetenv("RELAY_POOL_URL")
	os.Unsetenv("RELAY_SAMPLE_COUNT")
	os.Unsetenv("RELAY_TEST_COUNT")
	os.Unsetenv("RELAY_RANDOM_COUNT")

	// Default when unset should be 10
	stDefault := New(&Config{})
	if stDefault.Config().RelaySampleCount != 10 {
		t.Errorf("expected default RelaySampleCount 10, got %d", stDefault.Config().RelaySampleCount)
	}

	// Set RELAY_SAMPLE_COUNT=5
	os.Setenv("RELAY_SAMPLE_COUNT", "5")
	stSample := New(&Config{})
	if stSample.Config().RelaySampleCount != 5 {
		t.Errorf("expected RelaySampleCount 5 from env, got %d", stSample.Config().RelaySampleCount)
	}

	// Set RELAY_SAMPLE_COUNT="all"
	os.Setenv("RELAY_SAMPLE_COUNT", "all")
	stAll := New(&Config{})
	if stAll.Config().RelaySampleCount != -1 {
		t.Errorf("expected RelaySampleCount -1 for 'all', got %d", stAll.Config().RelaySampleCount)
	}

	os.Unsetenv("RELAY_SAMPLE_COUNT")
}
