package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/sindhu/unimark/pkg/bench"
	"github.com/sindhu/unimark/pkg/metrics"
	"github.com/sindhu/unimark/pkg/report"
	"github.com/sindhu/unimark/pkg/target"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run benchmarks across all targets",
	Example: `  unimark run --workload ./workloads/go-http
  unimark run --targets docker,nanos --verbose
  unimark run --runs 5 --output json
  unimark run --nanos-smp 8
  unimark run --nanos-version 2e85d1e`,
	RunE: runBenchmark,
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringP("workload", "w", "./workloads/go-http", "path to workload")
	runCmd.Flags().StringSliceP("targets", "t", []string{"docker", "nanos"}, "targets to benchmark")
	runCmd.Flags().IntP("runs", "r", 5, "number of measured runs per target")
	runCmd.Flags().IntP("warmup", "u", 1, "number of warmup runs to discard")
	runCmd.Flags().IntP("port", "p", 8080, "port the workload listens on")
	runCmd.Flags().DurationP("timeout", "T", 5*time.Minute, "timeout per run")
	runCmd.Flags().IntP("connections", "c", 50, "concurrent connections for load generator")
	runCmd.Flags().DurationP("duration", "d", 30*time.Second, "load generator duration")
	runCmd.Flags().BoolP("verbose", "v", false, "show boot phase breakdown")
	runCmd.Flags().Int("nanos-smp", 1, "number of vCPUs for Nanos (default 1, set to match Docker cores for fair CPU comparison)")
	runCmd.Flags().String("nanos-version", "", "Nanos kernel version or commit hash to use (e.g. 2e85d1e)")
}

func runBenchmark(cmd *cobra.Command, args []string) error {
	workloadPath, _ := cmd.Flags().GetString("workload")
	targetNames, _ := cmd.Flags().GetStringSlice("targets")
	runs, _ := cmd.Flags().GetInt("runs")
	warmup, _ := cmd.Flags().GetInt("warmup")
	port, _ := cmd.Flags().GetInt("port")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	connections, _ := cmd.Flags().GetInt("connections")
	duration, _ := cmd.Flags().GetDuration("duration")
	outputFormat, _ := cmd.Flags().GetString("output")
	outputFile, _ := cmd.Flags().GetString("file")
	verbose, _ := cmd.Flags().GetBool("verbose")
	nanosSmp, _ := cmd.Flags().GetInt("nanos-smp")
	nanosVersion, _ := cmd.Flags().GetString("nanos-version")

	workload := target.Workload{
		Name: "go-http",
		Port: port,
		Path: workloadPath,
	}

	targets, err := buildTargets(targetNames, nanosSmp, nanosVersion)
	if err != nil {
		return err
	}

	config := bench.Config{
		Runs:       runs,
		WarmupRuns: warmup,
		RunTimeout: timeout,
		Verbose:    verbose,
		Load: bench.LoadConfig{
			Duration:    duration,
			Connections: connections,
		},
	}

	ctx := context.Background()

	env, err := metrics.CaptureEnvironment(ctx)
	if err != nil {
		return fmt.Errorf("failed to capture environment: %w", err)
	}
	fmt.Printf("host: %s | arch: %s | kernel: %s\n\n",
		env.OS,
		env.Arch,
		env.KernelVersion,
	)

	runner := bench.NewRunner(config, targets)
	results, err := runner.Run(ctx, workload)
	if err != nil {
		return fmt.Errorf("benchmark failed: %w", err)
	}

	for i := range results {
		results[i].Metadata.KernelVersion = env.KernelVersion
		results[i].Metadata.CPU = env.CPU
		results[i].Metadata.HostMemory = env.HostMemory
	}

	renderer, err := buildRenderer(outputFormat, outputFile, verbose)
	if err != nil {
		return err
	}

	return renderer.Render(results)
}

func buildTargets(names []string, nanosSmp int, nanosVersion string) ([]target.Target, error) {
	var targets []target.Target

	for _, name := range names {
		switch name {
		case "docker":
			targets = append(targets, &target.DockerTarget{})
		case "nanos":
			targets = append(targets, &target.NanosTarget{
				SMP:          nanosSmp,
				NanosVersion: nanosVersion,
			})
		default:
			return nil, fmt.Errorf("unknown target: %s (valid: docker, nanos)", name)
		}
	}

	return targets, nil
}

func buildRenderer(format string, file string, verbose bool) (report.Renderer, error) {
	switch format {
	case "json":
		return report.NewJSONRenderer(file), nil
	case "markdown":
		return report.NewMarkdownRenderer(file), nil
	case "table", "":
		return report.NewTableRenderer(verbose), nil
	default:
		return nil, fmt.Errorf("unknown output format: %s (valid: table, json, markdown)", format)
	}
}
