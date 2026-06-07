package target

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type DockerTarget struct {
	workload    Workload
	containerID string
	endpoint    string
	imageBuilt  bool
}

func (d *DockerTarget) Name() string {
	return "docker"
}

func (d *DockerTarget) Setup(ctx context.Context, workload Workload) error {
	d.workload = workload
	d.endpoint = fmt.Sprintf("http://localhost:%d", workload.Port)

	cmd := exec.CommandContext(ctx, "docker", "build",
		"-t", workload.Name,
		workload.Path,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker build failed: %w\n%s", err, output)
	}
	d.imageBuilt = true
	return nil
}

func (d *DockerTarget) Start(ctx context.Context) (time.Duration, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, "docker", "run",
		"--detach",
		"--publish", fmt.Sprintf("%d:%d", d.workload.Port, d.workload.Port),
		"--name", d.workload.Name,
		d.workload.Name,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("docker run failed: %w\n%s", err, output)
	}
	d.containerID = strings.TrimSpace(string(output))

	if err := d.waitUntilReady(ctx); err != nil {
		return 0, err
	}
	return time.Since(start), nil
}

func (d *DockerTarget) Stop(ctx context.Context) error {
	if d.containerID == "" {
		return nil
	}

	// stop and remove container only — keep the image for next run
	cmd := exec.CommandContext(ctx, "docker", "stop", d.containerID)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker stop failed: %w\n%s", err, output)
	}

	cmd = exec.CommandContext(ctx, "docker", "rm", d.containerID)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker rm failed: %w\n%s", err, output)
	}

	d.containerID = ""
	return nil
}

func (d *DockerTarget) Teardown(ctx context.Context) error {
	// remove image only at the very end
	if d.imageBuilt {
		cmd := exec.CommandContext(ctx, "docker", "rmi", d.workload.Name)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("docker rmi failed: %w\n%s", err, output)
		}
		d.imageBuilt = false
	}
	return nil
}

func (d *DockerTarget) Endpoint() string {
	return d.endpoint
}

func (d *DockerTarget) waitUntilReady(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for container: %w", ctx.Err())
		default:
			resp, err := http.Get(d.endpoint)
			if err == nil {
				resp.Body.Close()
				return nil
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}
