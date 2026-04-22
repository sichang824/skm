package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultDBDSNUsesDotSKMOutsideRepo(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	previousWD := mustGetwd(t)
	t.Cleanup(func() {
		if err := chdir(previousWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	workspace := t.TempDir()
	if err := chdir(workspace); err != nil {
		t.Fatalf("change working directory: %v", err)
	}

	got := defaultDBDSN()
	want := filepath.Join(tempHome, ".skm", "app.db")
	if got != want {
		t.Fatalf("defaultDBDSN() = %q, want %q", got, want)
	}
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	return wd
}

var (
	getwd = func() (string, error) { return os.Getwd() }
	chdir = func(dir string) error { return os.Chdir(dir) }
)
