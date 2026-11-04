package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunTCPResourceOptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	if err := os.WriteFile(path, []byte(`{"type":"client","listen":"127.0.0.1:0","server":"127.0.0.1","target":"127.0.0.1:9","password":"local-test-password","nolog":true,"quiet":true,"tcp_buffer_bytes":-1,"tcp_window_frames":512}`), 0600); err != nil {
		t.Fatal(err)
	}
	var code int
	out := captureStdout(t, func() { code = run([]string{"-c", path}, closedStop()) })
	if code != 2 || !strings.Contains(out, "invalid TCP resources") {
		t.Fatalf("invalid config: code=%d output=%s", code, out)
	}
	// Explicit zero selects historical defaults even when the file contains
	// an invalid override; explicit valid values select the bounded profile.
	for _, value := range []string{"0", "262144"} {
		out = captureStdout(t, func() {
			code = run([]string{"-c", path, "-tcp-buffer-bytes", value, "-tcp-window-frames", "512"}, closedStop())
		})
		if code != 0 || strings.Contains(out, "invalid TCP resources") {
			t.Fatalf("CLI override %s: code=%d output=%s", value, code, out)
		}
	}
}
