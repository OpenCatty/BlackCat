package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/startower-observability/blackcat/internal/security"
)

const (
	defaultExecTimeout  = 60 * time.Second
	defaultMaxOutput    = 1 << 20 // 1 MB
	execToolName        = "exec"
	execToolDescription = "Execute a shell command on the server"
)

var execToolParameters = json.RawMessage(`{
	"type": "object",
	"properties": {
		"command": {
			"type": "string",
			"description": "Shell command to execute"
		},
		"workdir": {
			"type": "string",
			"description": "Working directory (optional)"
		}
	},
	"required": ["command"]
}`)

// ExecTool executes shell commands with deny-list filtering.
type ExecTool struct {
	denyList  *security.DenyList
	workDir   string
	timeout   time.Duration
	maxOutput int
}

// NewExecTool creates an ExecTool with the given deny list and defaults.
func NewExecTool(denyList *security.DenyList, workDir string, timeout time.Duration) *ExecTool {
	if timeout <= 0 {
		timeout = defaultExecTimeout
	}
	return &ExecTool{
		denyList:  denyList,
		workDir:   workDir,
		timeout:   timeout,
		maxOutput: defaultMaxOutput,
	}
}

func (t *ExecTool) Name() string                { return execToolName }
func (t *ExecTool) Description() string         { return execToolDescription }
func (t *ExecTool) Parameters() json.RawMessage { return execToolParameters }

// Execute runs a shell command after checking the deny list.
func (t *ExecTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Command string `json:"command"`
		Workdir string `json:"workdir"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("exec: invalid arguments: %w", err)
	}
	if params.Command == "" {
		return "", fmt.Errorf("exec: command is required")
	}

	// Check against deny list.
	if err := t.denyList.Check(params.Command); err != nil {
		return "", err
	}

	// Build the command with a timeout-scoped context.
	timeoutCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(timeoutCtx, "cmd", "/C", params.Command)
	} else {
		cmd = exec.CommandContext(timeoutCtx, "sh", "-c", params.Command)
	}

	// WaitDelay ensures the process is killed promptly after context cancellation.
	cmd.WaitDelay = 2 * time.Second

	// Set working directory.
	if params.Workdir != "" {
		cmd.Dir = params.Workdir
	} else if t.workDir != "" {
		cmd.Dir = t.workDir
	}

	output, err := cmd.CombinedOutput()

	// Truncate if output exceeds maxOutput.
	if len(output) > t.maxOutput {
		output = append(output[:t.maxOutput], []byte("\n... (output truncated)")...)
	}

	// Check if the context deadline was exceeded (timeout).
	if timeoutCtx.Err() != nil {
		return string(output), fmt.Errorf("exec: %w", timeoutCtx.Err())
	}

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return string(output), fmt.Errorf("exec: %w", err)
		}
	}

	return fmt.Sprintf("%s\n[exit code: %d]", string(output), exitCode), nil
}
