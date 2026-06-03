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

func TestRevealInFinderRequiresExistingPath(t *testing.T) {
	service := &DesktopService{
		revealPath: func(path string) error { return nil },
	}

	_, err := service.RevealInFinder("/does/not/exist")
	if err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestRevealInFinderRejectsEmptyPath(t *testing.T) {
	service := &DesktopService{
		revealPath: func(path string) error { return nil },
	}

	_, err := service.RevealInFinder("   ")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestRevealInFinderCallsRevealFn(t *testing.T) {
	dir := t.TempDir()
	var calledPath string
	service := &DesktopService{
		revealPath: func(path string) error {
			calledPath = path
			return nil
		},
	}

	result, err := service.RevealInFinder(dir)
	if err != nil {
		t.Fatalf("RevealInFinder returned error: %v", err)
	}
	if calledPath != dir {
		t.Fatalf("unexpected reveal path: got %s want %s", calledPath, dir)
	}
	if result.Path != dir {
		t.Fatalf("unexpected result path: got %s want %s", result.Path, dir)
	}
}
