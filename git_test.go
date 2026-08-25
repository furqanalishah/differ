package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadGitDiffIncludesTrackedAndUntrackedChanges(t *testing.T) {
	directory := t.TempDir()
	runTestGit(t, directory, "init", "--quiet")
	runTestGit(t, directory, "config", "user.email", "differ@example.com")
	runTestGit(t, directory, "config", "user.name", "Differ Test")

	trackedPath := filepath.Join(directory, "tracked.txt")
	if err := os.WriteFile(trackedPath, []byte("before\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, directory, "add", "tracked.txt")
	runTestGit(t, directory, "commit", "--quiet", "-m", "initial")

	if err := os.WriteFile(trackedPath, []byte("after\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "untracked.txt"), []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	output, err := loadGitDiff(context.Background(), directory, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "tracked.txt") || !strings.Contains(output, "-before") || !strings.Contains(output, "+after") {
		t.Fatalf("tracked change missing from diff:\n%s", output)
	}
	if !strings.Contains(output, "untracked.txt") || !strings.Contains(output, "+new") {
		t.Fatalf("untracked change missing from diff:\n%s", output)
	}
}

func TestLoadGitDiffRejectsNonRepository(t *testing.T) {
	_, err := loadGitDiff(context.Background(), t.TempDir(), nil)
	if err == nil || err.Error() != "not inside a Git repository" {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func TestLoadGitDiffWorksFromRepositorySubdirectory(t *testing.T) {
	directory := t.TempDir()
	runTestGit(t, directory, "init", "--quiet")
	runTestGit(t, directory, "config", "user.email", "differ@example.com")
	runTestGit(t, directory, "config", "user.name", "Differ Test")

	trackedPath := filepath.Join(directory, "root.txt")
	if err := os.WriteFile(trackedPath, []byte("before\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, directory, "add", "root.txt")
	runTestGit(t, directory, "commit", "--quiet", "-m", "initial")
	if err := os.WriteFile(trackedPath, []byte("after\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	subdirectory := filepath.Join(directory, "nested")
	if err := os.Mkdir(subdirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdirectory, "new.txt"), []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	output, err := loadGitDiff(context.Background(), subdirectory, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "root.txt") || !strings.Contains(output, "nested/new.txt") {
		t.Fatalf("repository-wide changes missing from nested invocation:\n%s", output)
	}
}

func runTestGit(t *testing.T, directory string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}
