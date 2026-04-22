package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindCLISourcePrefersBundledCLI(t *testing.T) {
	baseDir := t.TempDir()
	bundleRoot := filepath.Join(baseDir, "SKM.app", "Contents")
	executablePath := filepath.Join(bundleRoot, "MacOS", "SKM")
	bundledCLIPath := filepath.Join(bundleRoot, "Resources", "skm")
	if err := os.MkdirAll(filepath.Dir(executablePath), 0o755); err != nil {
		t.Fatalf("mkdir executable dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(bundledCLIPath), 0o755); err != nil {
		t.Fatalf("mkdir bundled cli dir: %v", err)
	}
	if err := os.WriteFile(executablePath, []byte("app"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}
	if err := os.WriteFile(bundledCLIPath, []byte("cli"), 0o755); err != nil {
		t.Fatalf("write bundled cli: %v", err)
	}

	service := &DesktopService{
		executablePath: func() (string, error) { return executablePath, nil },
		workingDir:     func() (string, error) { return baseDir, nil },
	}

	sourcePath, err := service.findCLISource()
	if err != nil {
		t.Fatalf("findCLISource returned error: %v", err)
	}
	if sourcePath != bundledCLIPath {
		t.Fatalf("unexpected source path: got %s want %s", sourcePath, bundledCLIPath)
	}
}

func TestFindCLISourceDoesNotUseDesktopAppBinary(t *testing.T) {
	baseDir := t.TempDir()
	bundleRoot := filepath.Join(baseDir, "SKM.app", "Contents")
	executablePath := filepath.Join(bundleRoot, "MacOS", "SKM")
	if err := os.MkdirAll(filepath.Dir(executablePath), 0o755); err != nil {
		t.Fatalf("mkdir executable dir: %v", err)
	}
	if err := os.WriteFile(executablePath, []byte("app"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}

	service := &DesktopService{
		executablePath: func() (string, error) { return executablePath, nil },
		workingDir:     func() (string, error) { return baseDir, nil },
	}

	sourcePath, err := service.findCLISource()
	if err != nil {
		t.Fatalf("findCLISource returned error: %v", err)
	}
	if sourcePath != "" {
		t.Fatalf("expected no CLI source, got %s", sourcePath)
	}
}
