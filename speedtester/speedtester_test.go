package speedtester

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCleanYAMLControlCharacters(t *testing.T) {
	// Raw text containing illegal YAML control characters:
	// \x1b (ESC), \x08 (BS), \x00 (NUL), \x7F (DEL), and C1 control \xC2\x90
	badYAML := "proxies:\n  - name: \x1b[31mNode\x00Name\x08Test\x7F\xC2\x90\n    type: ss\n    server: 1.1.1.1\n    port: 8388\n    cipher: aes-128-gcm\n    password: pwd\n"

	// Directly unmarshaling badYAML must fail with yaml: control characters are not allowed
	var m map[string]any
	err := yaml.Unmarshal([]byte(badYAML), &m)
	if err == nil {
		t.Fatalf("expected unmarshal error on bad YAML with control characters, got nil")
	}

	// After cleaning, unmarshal must succeed
	cleaned := CleanYAMLControlCharacters([]byte(badYAML))
	var mClean map[string]any
	if err := yaml.Unmarshal(cleaned, &mClean); err != nil {
		t.Fatalf("unmarshal should succeed after cleaning, got error: %v", err)
	}

	// FilterValidConfigs should also succeed without error
	filtered, err := FilterValidConfigs([]byte(badYAML))
	if err != nil {
		t.Fatalf("FilterValidConfigs failed on bad YAML: %v", err)
	}
	if len(filtered) == 0 {
		t.Fatalf("expected filtered output, got empty")
	}
}

func TestConvertMappedIPv6ToIPv4(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "1.1.1.1", expected: "1.1.1.1"},
		{input: "::ffff:192.0.2.1", expected: "192.0.2.1"},
		{input: "::ffff:127.0.0.1", expected: "127.0.0.1"},
		{input: "2001:db8::1", expected: "2001:db8::1"},
		{input: "example.com", expected: "example.com"},
		{input: "", expected: ""},
	}

	for _, tt := range tests {
		got := convertMappedIPv6ToIPv4(tt.input)
		if got != tt.expected {
			t.Errorf("convertMappedIPv6ToIPv4(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestLoadProxies_NonStringServer(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
proxies:
  - name: normal-ss
    type: ss
    server: 1.1.1.1
    port: 8388
    cipher: aes-128-gcm
    password: pwd
  - name: mapped-ipv6-ss
    type: ss
    server: "::ffff:1.2.3.4"
    port: 8388
    cipher: aes-128-gcm
    password: pwd
  - name: numeric-server-proxy
    type: socks5
    server: 12345.67
    port: 1080
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	st := New(&Config{
		ConfigPaths: configPath,
	})

	// Should not panic even when proxies contain non-string / float64 server values
	proxies, err := st.LoadProxies(false)
	if err != nil {
		t.Fatalf("LoadProxies returned unexpected error: %v", err)
	}

	// Verify mapped IPv6 was converted
	if p, ok := proxies["mapped-ipv6-ss"]; ok {
		if server, ok := p.Config["server"].(string); !ok || server != "1.2.3.4" {
			t.Errorf("expected server to be converted to 1.2.3.4, got %v", p.Config["server"])
		}
	} else {
		t.Errorf("expected mapped-ipv6-ss to be loaded")
	}
}

func TestFilterValidConfigs_NonStringServer(t *testing.T) {
	yamlContent := `
proxies:
  - name: normal-ss
    type: ss
    server: 1.1.1.1
    port: 8388
    cipher: aes-128-gcm
    password: pwd
  - name: mapped-ipv6-ss
    type: ss
    server: "::ffff:1.2.3.4"
    port: 8388
    cipher: aes-128-gcm
    password: pwd
  - name: numeric-server-proxy
    type: socks5
    server: 12345.67
    port: 1080
`
	filtered, err := FilterValidConfigs([]byte(yamlContent))
	if err != nil {
		t.Fatalf("FilterValidConfigs failed: %v", err)
	}
	if len(filtered) == 0 {
		t.Fatalf("expected filtered output, got empty")
	}
}
