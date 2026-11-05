package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPaddingOptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	if err := os.WriteFile(path, []byte(`{"type":"client","listen":"127.0.0.1:0","server":"127.0.0.1","target":"127.0.0.1:9","password":"local-test-password","nolog":true,"quiet":true,"padding_max":65}`), 0600); err != nil {
		t.Fatal(err)
	}
	var code int
	out := captureStdout(t, func() { code = run([]string{"-c", path}, closedStop()) })
	if code != 2 || !strings.Contains(out, "padding-max") {
		t.Fatalf("invalid config: %d %s", code, out)
	}
	for _, value := range []string{"0", "32"} {
		out = captureStdout(t, func() { code = run([]string{"-c", path, "-padding-max", value}, closedStop()) })
		if code != 0 || strings.Contains(out, "padding-max must") {
			t.Fatalf("override: %d %s", code, out)
		}
	}
}
