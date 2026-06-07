package target

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"time"
)

type NanosTarget struct {
	workload Workload
	process  *exec.Cmd
	endpoint string
}

func (n *NanosTarget) Setup(ctx context.Context, workload Workload) error {
	n.workload = workload
	n.endpoint = fmt.Sprintf("http://localhost:%d", workload.Port)
	return nil
}

func (n *NanosTarget) Start(ctx context.Context) (time.Duration, error) {
	start := time.Now()

	// ops run takes the binary directly and runs it as a unikernel
	cmd := exec.CommandContext(ctx, "ops", "run",
		n.workload.Path+"/server",
		"-p", fmt.Sprintf("%d", n.workload.Port),
		"-c", n.workload.Path+"/config.json",
	)

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("ops run failed: %w", err)
	}
	n.process = cmd

	if err := n.waitUntilReady(ctx); err != nil {
		return 0, err
	}

	return time.Since(start), nil
}

func (n *NanosTarget) Stop(ctx context.Context) error {
	if n.process == nil {
		return nil
	}

	if err := n.process.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill nanos process: %w", err)
	}

	n.process.Wait()
	n.process = nil
	return nil
}

func (n *NanosTarget) Endpoint() string {
	return n.endpoint
}

func (n *NanosTarget) waitUntilReady(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for nanos: %w", ctx.Err())
		default:
			resp, err := http.Get(n.endpoint)
			if err == nil {
				resp.Body.Close()
				return nil
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (n *NanosTarget) Name() string {
	return "nanos"
}
