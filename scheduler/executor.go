package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"

	"github.com/startower-observability/blackcat/config"
)

// JobExecutor defines how scheduled jobs execute their commands.
type JobExecutor interface {
	Execute(ctx context.Context, job config.ScheduledJob) error
}

// ShellExecutor runs job commands as shell subprocesses.
type ShellExecutor struct{}

// Execute runs the job's Command as a shell subprocess.
func (e *ShellExecutor) Execute(ctx context.Context, job config.ScheduledJob) error {
	if job.Command == "" {
		return fmt.Errorf("job %q: command is empty", job.Name)
	}

	cmd := shellCommand(ctx, job.Command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	slog.Info("executing scheduled job", "job", job.Name, "command", job.Command)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("job %q command failed: %w", job.Name, err)
	}

	return nil
}

// shellCommand returns an exec.Cmd appropriate for the current OS.
func shellCommand(ctx context.Context, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/C", command)
	}
	return exec.CommandContext(ctx, "sh", "-c", command)
}
