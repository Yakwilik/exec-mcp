package handler

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	execmcpv1 "github.com/Yakwilik/exec-mcp/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handler implements the generated ExecMcpAPIToolHandler interface.
type Handler struct{}

// New creates a new Handler.
func New() *Handler {
	return &Handler{}
}

// ExecProcess starts a background process and returns its PID.
func (h *Handler) ExecProcess(_ context.Context, req *execmcpv1.ExecProcessRequest) (*execmcpv1.ExecProcessResponse, error) {
	cmd := exec.Command(req.Command, req.Args...)

	if req.Dir != nil && *req.Dir != "" {
		cmd.Dir = *req.Dir
	}

	if len(req.Env) > 0 {
		cmd.Env = append(os.Environ(), req.Env...)
	}

	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("error starting process: %w", err)
	}

	return &execmcpv1.ExecProcessResponse{
		Pid:       int32(cmd.Process.Pid),
		Command:   req.Command,
		Args:      req.Args,
		StartTime: timestamppb.New(startTime),
		Status:    "running",
	}, nil
}

// RunCommand executes a command synchronously and returns its output.
func (h *Handler) RunCommand(ctx context.Context, req *execmcpv1.RunCommandRequest) (*execmcpv1.RunCommandResponse, error) {
	execCtx := ctx
	var cancel context.CancelFunc

	if req.Timeout != nil && *req.Timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, time.Duration(*req.Timeout)*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(execCtx, req.Command, req.Args...)

	if req.Dir != nil && *req.Dir != "" {
		cmd.Dir = *req.Dir
	}

	if len(req.Env) > 0 {
		cmd.Env = append(os.Environ(), req.Env...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	runErr := cmd.Run()
	elapsed := time.Since(startTime)
	executedAt := startTime.Add(elapsed)

	exitCode := int32(0)
	success := true

	if runErr != nil {
		success = false
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = int32(exitErr.ExitCode())
		} else {
			exitCode = -1
		}
	}

	return &execmcpv1.RunCommandResponse{
		Command:         req.Command,
		Args:            req.Args,
		ExitCode:        exitCode,
		Stdout:          stdout.String(),
		Stderr:          stderr.String(),
		ExecutedAt:      timestamppb.New(executedAt),
		DurationSeconds: elapsed.Seconds(),
		Success:         success,
	}, nil
}

// StopProcess sends a signal to the process identified by pid.
func (h *Handler) StopProcess(_ context.Context, req *execmcpv1.StopProcessRequest) (*execmcpv1.StopProcessResponse, error) {
	process, err := os.FindProcess(int(req.Pid))
	if err != nil {
		return nil, fmt.Errorf("error finding process %d: %w", req.Pid, err)
	}

	useKill := req.Kill != nil && *req.Kill

	var signal syscall.Signal
	var signalName string

	if useKill {
		signal = syscall.SIGKILL
		signalName = "SIGKILL"
	} else {
		signal = syscall.SIGTERM
		signalName = "SIGTERM"
	}

	if err := process.Signal(signal); err != nil {
		return nil, fmt.Errorf("error sending signal to process %d: %w", req.Pid, err)
	}

	return &execmcpv1.StopProcessResponse{
		Pid:    req.Pid,
		Signal: signalName,
		Status: "signal_sent",
	}, nil
}
