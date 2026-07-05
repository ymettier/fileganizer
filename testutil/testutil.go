package testutil

import (
	"os"
	"testing"
)

func UseTempDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Logf("failed to restore working directory: %v", err)
		}
	})
}
