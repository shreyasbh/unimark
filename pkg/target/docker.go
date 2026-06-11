package target

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type DockerTarget struct {
	workload    Workload
	containerID string
	endpoint    string
	imageBuilt  bool
	phases      BootPhases
}

func (d *DockerTarget) Name() string {
	return "docker"
}

func (d *DockerTarget) BootPhases() BootPhases {
	return d.phases
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

	// start docker events listener before launching container
	eventsCmd := exec.CommandContext(ctx, "docker", "events",
		"--filter", fmt.Sprintf("event=create"),
		"--filter", fmt.Sprintf("event=network"),
		"--filter", fmt.Sprintf("event=start"),
		"--format", "{{.Action}}",
	)
	eventsOut, err := eventsCmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("failed to get events pipe: %w", err)
	}
	eventsCmd.Start()

	// track phase timestamps
	var createTime, networkTime time.Time
	eventsDone := make(chan struct{})

	go func() {
		defer close(eventsDone)
		scanner := bufio.NewScanner(eventsOut)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			switch {
			case line == "create" && createTime.IsZero():
				createTime = time.Now()
			case line == "connect" && networkTime.IsZero():
				networkTime = time.Now()
			}
		}
	}()

	// launch container
	cmd := exec.CommandContext(ctx, "docker", "run",
		"--detach",
		"--publish", fmt.Sprintf("%d:%d", d.workload.Port, d.workload.Port),
		"--name", d.workload.Name,
		d.workload.Name,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		eventsCmd.Process.Kill()
		return 0, fmt.Errorf("docker run failed: %w\n%s", err, output)
	}
	d.containerID = strings.TrimSpace(string(output))

	// wait until app is ready
	if err := d.waitUntilReady(ctx); err != nil {
		eventsCmd.Process.Kill()
		return 0, err
	}

	total := time.Since(start)
	eventsCmd.Process.Kill()

	// calculate phases
	if !createTime.IsZero() {
		d.phases.ContainerCreate = createTime.Sub(start)
	}
	if !networkTime.IsZero() && !createTime.IsZero() {
		d.phases.NetworkConnect = networkTime.Sub(createTime)
	}
	d.phases.ProcessStart = total - d.phases.ContainerCreate - d.phases.NetworkConnect
	d.phases.Total = total

	return total, nil
}

func (d *DockerTarget) Stop(ctx context.Context) error {
	if d.containerID == "" {
		return nil
	}

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

func (d *DockerTarget) Memory(ctx context.Context) (int64, error) {
	if d.containerID == "" {
		return 0, fmt.Errorf("container not running")
	}

	cmd := exec.CommandContext(ctx, "docker", "stats",
		"--no-stream",
		"--format", "{{.MemUsage}}",
		d.containerID,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("docker stats failed: %w", err)
	}

	return parseDockerMemory(strings.TrimSpace(string(output)))
}

func parseDockerMemory(raw string) (int64, error) {
	parts := strings.Split(raw, " / ")
	if len(parts) != 2 {
		return 0, fmt.Errorf("unexpected memory format: %s", raw)
	}

	used := strings.TrimSpace(parts[0])

	var value float64
	var unit string
	_, err := fmt.Sscanf(used, "%f%s", &value, &unit)
	if err != nil {
		return 0, fmt.Errorf("could not parse memory value: %w", err)
	}

	switch strings.ToLower(unit) {
	case "mib":
		return int64(value * 1024 * 1024), nil
	case "gib":
		return int64(value * 1024 * 1024 * 1024), nil
	case "kib":
		return int64(value * 1024), nil
	default:
		return 0, fmt.Errorf("unknown memory unit: %s", unit)
	}
}

func (d *DockerTarget) PID(ctx context.Context) (int, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect",
		"--format", "{{.State.Pid}}",
		d.containerID,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("docker inspect failed: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0, fmt.Errorf("failed to parse PID: %w", err)
	}

	return pid, nil
}
