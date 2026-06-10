package target

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type NanosTarget struct {
	workload Workload
	process  *exec.Cmd
	endpoint string
	logFile  *os.File
	phases   BootPhases
}

func (n *NanosTarget) Name() string {
	return "nanos"
}

func (n *NanosTarget) BootPhases() BootPhases {
	return n.phases
}

// timestampRegex matches lines like "[0.004140] en1: assigned"
var timestampRegex = regexp.MustCompile(`\[(\d+\.\d+)\]`)

func (n *NanosTarget) Start(ctx context.Context) (time.Duration, error) {
	killPort(n.workload.Port)

	start := time.Now()

	logFile, err := os.CreateTemp("", "nanos-*.log")
	if err != nil {
		return 0, fmt.Errorf("failed to create log file: %w", err)
	}
	n.logFile = logFile

	cmd := exec.CommandContext(ctx, "ops", "run",
		n.workload.Path+"/server",
		"-p", fmt.Sprintf("%d", n.workload.Port),
		"-c", n.workload.Path+"/config.json",
	)

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("ops run failed: %w", err)
	}
	n.process = cmd

	ready := make(chan time.Duration, 1)

	go func() {
		f, err := os.Open(logFile.Name())
		if err != nil {
			return
		}
		defer f.Close()

		var firstLineTime time.Time
		var networkUpSecs float64

		scanner := bufio.NewScanner(f)
		for {
			for scanner.Scan() {
				line := scanner.Text()

				// capture time of first log line — QEMU is up
				if firstLineTime.IsZero() {
					firstLineTime = time.Now()
					n.phases.QEMUStartup = firstLineTime.Sub(start)
				}

				// capture network up timestamp from kernel log
				// line looks like "[0.004140] en1: assigned 10.0.2.15"
				if strings.Contains(line, "en1: assigned") && networkUpSecs == 0 {
					matches := timestampRegex.FindStringSubmatch(line)
					if len(matches) > 1 {
						networkUpSecs, _ = strconv.ParseFloat(matches[1], 64)
						n.phases.KernelBoot = time.Duration(networkUpSecs * float64(time.Second))
					}
				}

				// app ready
				if strings.Contains(line, "listening on") {
					appReady := time.Now()
					n.phases.NetworkUp = time.Duration(networkUpSecs * float64(time.Second))
					n.phases.AppReady = appReady.Sub(firstLineTime) - n.phases.KernelBoot
					n.phases.Total = time.Since(start)
					ready <- n.phases.Total
					return
				}
			}
			time.Sleep(10 * time.Millisecond)
			scanner = bufio.NewScanner(f)
		}
	}()

	select {
	case bootTime := <-ready:
		return bootTime, nil
	case <-ctx.Done():
		return 0, fmt.Errorf("timed out waiting for nanos: %w", ctx.Err())
	case <-time.After(30 * time.Second):
		return 0, fmt.Errorf("nanos did not start within 30 seconds")
	}
}

func (n *NanosTarget) Setup(ctx context.Context, workload Workload) error {
	n.workload = workload
	n.endpoint = fmt.Sprintf("http://localhost:%d", workload.Port)
	return nil
}

func (n *NanosTarget) Stop(ctx context.Context) error {
	if n.process != nil {
		n.process.Process.Kill()
		n.process.Wait()
		n.process = nil
	}

	killPort(n.workload.Port)

	if n.logFile != nil {
		os.Remove(n.logFile.Name())
		n.logFile = nil
	}

	return nil
}

func (n *NanosTarget) Endpoint() string {
	return n.endpoint
}

func (n *NanosTarget) PID() (int, error) {
	if n.process == nil {
		return 0, fmt.Errorf("nanos process not running")
	}
	return n.process.Process.Pid, nil
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
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func killPort(port int) {
	cmd := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port))
	output, err := cmd.CombinedOutput()
	if err != nil || len(strings.TrimSpace(string(output))) == 0 {
		return
	}
	for _, pid := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		exec.Command("kill", "-9", pid).Run()
	}
	time.Sleep(500 * time.Millisecond)
}

// Memory returns QEMU process memory from host perspective using ps
// works on both Mac and Linux
func (n *NanosTarget) Memory(ctx context.Context) (int64, error) {
	if n.process == nil {
		return 0, fmt.Errorf("nanos process not running")
	}

	pid := n.process.Process.Pid

	cmd := exec.CommandContext(ctx, "ps", "-o", "rss=", "-p",
		fmt.Sprintf("%d", pid),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("ps failed: %w", err)
	}

	// ps returns RSS in kilobytes
	kb, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse RSS: %w", err)
	}

	return kb * 1024, nil
}
