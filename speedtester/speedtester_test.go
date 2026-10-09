package speedtester

import (
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
