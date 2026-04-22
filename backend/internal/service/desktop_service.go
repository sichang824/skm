package service

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type DesktopService struct {
	executablePath func() (string, error)
	homeDir        func() (string, error)
	workingDir     func() (string, error)
}

type CLIInstallStatus struct {
	Available     bool   `json:"available"`
	Installed     bool   `json:"installed"`
	SourcePath    string `json:"sourcePath,omitempty"`
	InstalledPath string `json:"installedPath"`
}

type CLIInstallResult struct {
	SourcePath    string `json:"sourcePath"`
	InstalledPath string `json:"installedPath"`
	Replaced      bool   `json:"replaced"`
}

func NewDesktopService() *DesktopService {
	return &DesktopService{
		executablePath: os.Executable,
		homeDir:        os.UserHomeDir,
		workingDir:     os.Getwd,
	}
}

func (s *DesktopService) CLIStatus() (*CLIInstallStatus, error) {
	installedPath, err := s.cliInstallPath()
	if err != nil {
		return nil, err
	}
	sourcePath, _ := s.findCLISource()
	_, statErr := os.Stat(installedPath)
	return &CLIInstallStatus{
		Available:     sourcePath != "",
		Installed:     statErr == nil,
		SourcePath:    sourcePath,
		InstalledPath: installedPath,
	}, nil
}

func (s *DesktopService) InstallCLI() (*CLIInstallResult, error) {
	sourcePath, err := s.findCLISource()
	if err != nil {
		return nil, err
	}
	if sourcePath == "" {
		return nil, fmt.Errorf("desktop CLI binary is not available")
	}
	installedPath, err := s.cliInstallPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(installedPath), 0o755); err != nil {
		return nil, err
	}
	_, statErr := os.Stat(installedPath)
	replaced := statErr == nil
	if err := copyExecutableFile(sourcePath, installedPath); err != nil {
		return nil, err
	}
	return &CLIInstallResult{
		SourcePath:    sourcePath,
		InstalledPath: installedPath,
		Replaced:      replaced,
	}, nil
}

func (s *DesktopService) cliInstallPath() (string, error) {
	homeDir, err := s.homeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".local", "bin", "skm"), nil
}

func (s *DesktopService) findCLISource() (string, error) {
	for _, candidate := range s.cliSourceCandidates() {
		ok, err := exactRegularFileExists(candidate)
		if err != nil {
			return "", err
		}
		if ok {
			return candidate, nil
		}
	}
	return "", nil
}

func (s *DesktopService) cliSourceCandidates() []string {
	candidates := make([]string, 0, 12)
	if executablePath, err := s.executablePath(); err == nil {
		executableDir := filepath.Dir(executablePath)
		contentsDir := filepath.Dir(executableDir)
		candidates = append(candidates, filepath.Join(contentsDir, "Resources", "skm"))
		for current := executableDir; current != filepath.Dir(current); current = filepath.Dir(current) {
			candidates = append(candidates, filepath.Join(current, "build", "bin", "skm"))
		}
	}
	if wd, err := s.workingDir(); err == nil {
		for current := wd; current != filepath.Dir(current); current = filepath.Dir(current) {
			candidates = append(candidates, filepath.Join(current, "build", "bin", "skm"))
		}
	}
	return uniqueStrings(candidates)
}

func exactRegularFileExists(path string) (bool, error) {
	directory := filepath.Dir(path)
	base := filepath.Base(path)
	entries, err := os.ReadDir(directory)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	for _, entry := range entries {
		if entry.Name() != base {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return false, err
		}
		return !info.IsDir(), nil
	}
	return false, nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func copyExecutableFile(sourcePath, targetPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	info, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	tempPath := targetPath + ".tmp"
	targetFile, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		_ = targetFile.Close()
		return err
	}
	if err := targetFile.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tempPath, info.Mode()|0o111); err != nil {
		return err
	}
	return os.Rename(tempPath, targetPath)
}
