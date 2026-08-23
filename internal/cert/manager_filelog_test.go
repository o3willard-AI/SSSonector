package cert

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/o3willard-AI/SSSonector/internal/cert/generator"
	"go.uber.org/zap"
)

// TestCertLogsAreValidJSONOnStream verifies that certificate lifecycle
// messages survive as parseable JSON entries on a real zap output stream,
// matching what operators tail in production.
func TestCertLogsAreValidJSONOnStream(t *testing.T) {
	dir := t.TempDir()

	if err := generator.GenerateTemporaryCertificates(dir); err != nil {
		t.Fatalf("generate certificates: %v", err)
	}

	logPath := filepath.Join(dir, "cert-stream.log")

	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{logPath}
	cfg.Encoding = "json"
	logger, err := cfg.Build()
	if err != nil {
		t.Fatalf("build logger: %v", err)
	}

	mgr, err := NewManager(
		filepath.Join(dir, "server.crt"),
		filepath.Join(dir, "server.key"),
		filepath.Join(dir, "ca.crt"),
		true,
		false,
		logger,
	)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	defer mgr.Stop()

	mgr.SetRotationThreshold(0)
	drained := make(chan struct{})
	defer close(drained)
	go func() {
		for {
			select {
			case <-mgr.rotationDone:
			case <-drained:
				return
			}
		}
	}()

	mgr.checkCertificate()
	_ = logger.Sync()

	f, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("open log: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	foundStatus := false
	lines := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lines++
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("line %d is not valid JSON: %v\n%s", lines, err, line)
			continue
		}
		if msg, _ := entry["msg"].(string); msg == "Certificate status" {
			foundStatus = true
			if _, ok := entry["serial"]; !ok {
				t.Error("Certificate status entry missing serial field")
			}
			if _, ok := entry["expires_in"]; !ok {
				t.Error("Certificate status entry missing expires_in field")
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan log: %v", err)
	}
	if lines == 0 {
		t.Fatal("Expected log output, got empty file")
	}
	if !foundStatus {
		t.Error("Did not find a Certificate status entry in the stream")
	}
}
