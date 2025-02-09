package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDirs represents test directories
type TestDirs struct {
	Root   string // Root test directory
	Config string // Config directory
	Run    string // Run directory
	Log    string // Log directory
	Lib    string // Lib directory
}

// CreateTestDirs creates test directories
func CreateTestDirs(t *testing.T) *TestDirs {
	t.Helper()

	// Create temp root directory
	root, err := os.MkdirTemp("", "sssonector-test-*")
	require.NoError(t, err)

	// Create subdirectories
	dirs := &TestDirs{
		Root:   root,
		Config: filepath.Join(root, "etc", "sssonector"),
		Run:    filepath.Join(root, "var", "run"),
		Log:    filepath.Join(root, "var", "log"),
		Lib:    filepath.Join(root, "var", "lib", "sssonector"),
	}

	// Create all directories
	for _, dir := range []string{
		dirs.Config,
		dirs.Run,
		dirs.Log,
		dirs.Lib,
	} {
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err)
	}

	// Clean up on test completion
	t.Cleanup(func() {
		os.RemoveAll(root)
	})

	return dirs
}

// CreateTestUser creates a test user
func CreateTestUser(t *testing.T, username string) {
	t.Helper()

	// Skip if not root
	if os.Getuid() != 0 {
		t.Skip("Test requires root privileges")
	}

	// Create user
	cmd := exec.Command("useradd", "-r", "-s", "/sbin/nologin", username)
	err := cmd.Run()
	require.NoError(t, err)

	// Clean up on test completion
	t.Cleanup(func() {
		cmd := exec.Command("userdel", "-r", username)
		cmd.Run() // Ignore errors during cleanup
	})
}
