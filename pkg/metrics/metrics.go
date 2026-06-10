package metrics

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

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
	cmd := exec.Command("uname", "-m")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown", nil
	}
	return strings.TrimSpace(string(output)), nil
}

func totalMemory(ctx context.Context) (int64, error) {
	cmd := exec.CommandContext(ctx, "sysctl", "-n", "hw.memsize")
	output, err := cmd.CombinedOutput()
	if err != nil {
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

// ReadProcessMemory reads resident set size of a process in bytes
// reads from /proc/<pid>/status — works for any process on the host
func ReadProcessMemory(pid int) (int64, error) {
	path := fmt.Sprintf("/proc/%d/status", pid)
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("failed to read %s: %w", path, err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) != 3 {
				return 0, fmt.Errorf("unexpected VmRSS format: %s", line)
			}
			kb, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("failed to parse VmRSS: %w", err)
			}
			return kb * 1024, nil
		}
	}

	return 0, fmt.Errorf("VmRSS not found in %s", path)
}
