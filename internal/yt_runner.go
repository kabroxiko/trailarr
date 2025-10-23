package internal

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

// YtDlpRunner abstracts running yt-dlp so tests can inject a fake runner.
type YtDlpRunner interface {
	// StartCommand starts the command and returns a reader for stdout and the started *exec.Cmd.
	StartCommand(ctx context.Context, name string, args []string) (io.ReadCloser, *exec.Cmd, error)
	// CombinedOutput runs the command and returns combined stdout/stderr bytes.
	CombinedOutput(name string, args []string, dir string) ([]byte, error)
}

// DefaultYtDlpRunner uses os/exec to run yt-dlp.
type DefaultYtDlpRunner struct{}

func (r *DefaultYtDlpRunner) StartCommand(ctx context.Context, name string, args []string) (io.ReadCloser, *exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get StdoutPipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start %s: %w", name, err)
	}
	return stdout, cmd, nil
}

func (r *DefaultYtDlpRunner) CombinedOutput(name string, args []string, dir string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd.CombinedOutput()
}

// Package-level runner variable; tests can replace this with a fake implementation.
var ytDlpRunner YtDlpRunner = &DefaultYtDlpRunner{}
