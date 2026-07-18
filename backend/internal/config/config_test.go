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
	if err := os.Mkdir(filepath.Join(workspace, "data"), 0o755); err != nil {
		t.Fatalf("create unrelated data dir: %v", err)
	}
	if err := chdir(workspace); err != nil {
		t.Fatalf("change working directory: %v", err)
	}

	got := defaultDBDSN()
	want := filepath.Join(tempHome, ".skm", "app.db")
	if got != want {
		t.Fatalf("defaultDBDSN() = %q, want %q", got, want)
	}
}

func TestDefaultDBDSNUsesRepoDatabaseAtWorkspaceRoot(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	previousWD := mustGetwd(t)
	t.Cleanup(func() {
		if err := chdir(previousWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "backend", "data"), 0o755); err != nil {
		t.Fatalf("create backend data dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "wails.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write wails.json: %v", err)
	}
	if err := chdir(workspace); err != nil {
		t.Fatalf("change working directory: %v", err)
	}

	got := defaultDBDSN()
	if got != "./backend/data/app.db" {
		t.Fatalf("defaultDBDSN() = %q, want %q", got, "./backend/data/app.db")
	}
}

func TestDefaultDBDSNUsesRepoDatabaseAtBackendDir(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	previousWD := mustGetwd(t)
	t.Cleanup(func() {
		if err := chdir(previousWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	backendDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(backendDir, "data"), 0o755); err != nil {
		t.Fatalf("create data dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "go.mod"), []byte("module backend-go\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := chdir(backendDir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}

	got := defaultDBDSN()
	if got != "./data/app.db" {
		t.Fatalf("defaultDBDSN() = %q, want %q", got, "./data/app.db")
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

func TestLoadIgnoresUnrelatedProjectEnv(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	unsetEnv(t, "DB_DSN", "DB_DRIVER")

	unrelated := t.TempDir()
	envContent := "DB_DRIVER=postgres\nDB_DSN=postgres://evil@127.0.0.1:55433/bad?sslmode=disable\n"
	if err := os.WriteFile(filepath.Join(unrelated, ".env"), []byte(envContent), 0o644); err != nil {
		t.Fatalf("write unrelated .env: %v", err)
	}

	previousWD := mustGetwd(t)
	t.Cleanup(func() {
		if err := chdir(previousWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})
	if err := chdir(unrelated); err != nil {
		t.Fatalf("change working directory: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	wantDSN := filepath.Join(tempHome, ".skm", "app.db")
	if cfg.DBDriver != "sqlite" {
		t.Fatalf("DBDriver = %q, want sqlite", cfg.DBDriver)
	}
	if cfg.DBDSN != wantDSN {
		t.Fatalf("DBDSN = %q, want %q", cfg.DBDSN, wantDSN)
	}
}

func TestLoadUsesWorkspaceEnvInBackendDir(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	unsetEnv(t, "DB_DSN", "DB_DRIVER")

	backendDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(backendDir, "data"), 0o755); err != nil {
		t.Fatalf("create data dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "go.mod"), []byte("module backend-go\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	envContent := "DB_DRIVER=sqlite\nDB_DSN=./data/workspace.db\n"
	if err := os.WriteFile(filepath.Join(backendDir, ".env"), []byte(envContent), 0o644); err != nil {
		t.Fatalf("write backend .env: %v", err)
	}

	previousWD := mustGetwd(t)
	t.Cleanup(func() {
		if err := chdir(previousWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})
	if err := chdir(backendDir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	wantDSN := filepath.Join("data", "workspace.db")
	if cfg.DBDriver != "sqlite" {
		t.Fatalf("DBDriver = %q, want sqlite", cfg.DBDriver)
	}
	if cfg.DBDSN != wantDSN {
		t.Fatalf("DBDSN = %q, want %q", cfg.DBDSN, wantDSN)
	}
}

func unsetEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}
}

var (
	getwd = func() (string, error) { return os.Getwd() }
	chdir = func(dir string) error { return os.Chdir(dir) }
)
