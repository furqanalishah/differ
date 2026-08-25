package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func loadGitDiff(ctx context.Context, directory string, userArgs []string) (string, error) {
	repositoryRoot, err := runGit(ctx, directory, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", errors.New("not inside a Git repository")
	}
	repositoryRoot = strings.TrimSpace(repositoryRoot)

	if len(userArgs) > 0 {
		output, err := runGitDiff(ctx, directory, userArgs...)
		if err != nil {
			return "", err
		}
		return output, nil
	}

	args := []string{"HEAD"}
	if _, err := runGit(ctx, directory, "rev-parse", "--verify", "HEAD"); err != nil {
		args = []string{"--cached"}
	}

	tracked, err := runGitDiff(ctx, directory, args...)
	if err != nil {
		return "", err
	}

	untracked, err := untrackedDiffs(ctx, repositoryRoot)
	if err != nil {
		return "", err
	}

	return joinDiffs(tracked, untracked), nil
}

func runGitDiff(ctx context.Context, directory string, args ...string) (string, error) {
	gitArgs := []string{"-c", "color.ui=false", "diff", "--no-ext-diff", "--no-color", "--find-renames"}
	gitArgs = append(gitArgs, args...)
	return runGit(ctx, directory, gitArgs...)
}

func untrackedDiffs(ctx context.Context, directory string) (string, error) {
	output, err := runGitBytes(ctx, directory, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return "", err
	}

	var result strings.Builder
	for _, path := range bytes.Split(output, []byte{0}) {
		if len(path) == 0 {
			continue
		}

		diff, diffErr := runGitDiff(ctx, directory, "--no-index", "--", "/dev/null", string(path))
		if diffErr != nil {
			return "", diffErr
		}
		result.WriteString(diff)
		if !strings.HasSuffix(diff, "\n") {
			result.WriteByte('\n')
		}
	}
	return result.String(), nil
}

func joinDiffs(parts ...string) string {
	var result strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		if result.Len() > 0 && !strings.HasSuffix(result.String(), "\n") {
			result.WriteByte('\n')
		}
		result.WriteString(part)
	}
	return result.String()
}

func runGit(ctx context.Context, directory string, args ...string) (string, error) {
	output, err := runGitBytes(ctx, directory, args...)
	return string(output), err
}

func runGitBytes(ctx context.Context, directory string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = directory
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err == nil {
		return output, nil
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 1 && len(output) > 0 {
		return output, nil
	}

	message := strings.TrimSpace(stderr.String())
	if message == "" {
		message = err.Error()
	}
	return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), message)
}
