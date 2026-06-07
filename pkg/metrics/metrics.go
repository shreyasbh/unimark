package metrics

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type LoadResult struct {
	Throughput float64
	LatencyP50 float64
	LatencyP95 float64
	LatencyP99 float64
}

type Environment struct {
	KernelVersion string
	CPU           string
	HostMemory    int64
	OS            string
	Arch          string
}

func CaptureEnvironment(ctx context.Context) (Environment, error) {
	cpu, err := cpuModel()
	if err != nil {
		return Environment{}, fmt.Errorf("failed to get CPU model: %w", err)
	}

	mem, err := totalMemory(ctx)
	if err != nil {
		return Environment{}, fmt.Errorf("failed to get total memory: %w", err)
	}

	return Environment{
		KernelVersion: kernelVersion(ctx),
		CPU:           cpu,
		HostMemory:    mem,
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
	}, nil
}

func kernelVersion(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "uname", "-r")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

func cpuModel() (string, error) {
	// works on both Linux and Mac
	cmd := exec.Command("uname", "-m")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown", nil
	}
	return strings.TrimSpace(string(output)), nil
}

func totalMemory(ctx context.Context) (int64, error) {
	// sysctl works on Mac and Linux
	cmd := exec.CommandContext(ctx, "sysctl", "-n", "hw.memsize")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// fallback for Linux
		cmd = exec.CommandContext(ctx, "grep", "MemTotal", "/proc/meminfo")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return 0, fmt.Errorf("failed to get memory: %w", err)
		}
		fields := strings.Fields(string(output))
		if len(fields) < 2 {
			return 0, fmt.Errorf("unexpected meminfo format")
		}
		kb, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return 0, err
		}
		return kb * 1024, nil
	}

	bytes, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return 0, err
	}
	return bytes, nil
}
