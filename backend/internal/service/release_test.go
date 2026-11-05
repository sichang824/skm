package service

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestReleaseFindAndInstallPortableCLI(t *testing.T) {
	root := t.TempDir()
	cli := filepath.Join(root, "cli", cliBinaryName)
	if err := os.MkdirAll(filepath.Dir(cli), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cli, []byte("packaged-cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	s := &DesktopService{
		executablePath: func() (string, error) { return filepath.Join(root, "SKM.exe"), nil },
		workingDir:     func() (string, error) { return root, nil },
		homeDir:        func() (string, error) { return home, nil },
	}
	for i := 0; i < 2; i++ {
		result, err := s.InstallCLI()
		if err != nil {
			t.Fatal(err)
		}
		if result.Replaced != (i == 1) {
			t.Fatalf("unexpected replacement state: %+v", result)
		}
		content, err := os.ReadFile(filepath.Join(home, ".local", "bin", cliBinaryName))
		if err != nil || string(content) != "packaged-cli" {
			t.Fatalf("installed CLI: %q, %v", content, err)
		}
	}
}

func TestReleaseTimeoutKillsChildProcesses(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("Node.js is required")
	}
	root := t.TempDir()
	script := `const {spawn} = require('node:child_process');
spawn(process.execPath, ['-e', 'const fs=require("node:fs"); setInterval(()=>fs.appendFileSync("heartbeat", "x"), 20)'], {stdio:'inherit'});
setInterval(()=>{}, 1000);`
	if err := os.WriteFile(filepath.Join(root, "parent.cjs"), []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := runLifecycleCommand(context.Background(), root, "node parent.cjs", os.Environ(), time.Second, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.TimedOut || result.ExitCode != 124 {
		t.Fatalf("timeout result: %+v", result)
	}
	heartbeat := filepath.Join(root, "heartbeat")
	before, err := os.ReadFile(heartbeat)
	if err != nil || len(before) == 0 {
		t.Fatalf("child did not start: %v", err)
	}
	time.Sleep(150 * time.Millisecond)
	after, err := os.ReadFile(heartbeat)
	if err != nil || len(after) != len(before) {
		t.Fatalf("child survived timeout: %d -> %d, %v", len(before), len(after), err)
	}
}
