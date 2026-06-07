package bench

import (
	"context"
	"fmt"
	"time"

	"github.com/sindhu/unimark/pkg/target"
)

type Config struct {
	Runs       int
	WarmupRuns int
	RunTimeout time.Duration
	Load       LoadConfig
}

type LoadConfig struct {
	Duration    time.Duration
	Connections int
	RateLimit   int
}

type Runner struct {
	config  Config
	targets []target.Target
}

func NewRunner(config Config, targets []target.Target) *Runner {
	return &Runner{
		config:  config,
		targets: targets,
	}
}

func (r *Runner) Run(ctx context.Context, workload target.Workload) ([]target.Result, error) {
	var results []target.Result

	for _, t := range r.targets {
		fmt.Printf("→ benchmarking %s\n", t.Endpoint())

		result, err := r.runTarget(ctx, t, workload)
		if err != nil {
			return nil, fmt.Errorf("target failed: %w", err)
		}

		results = append(results, result)
	}

	return results, nil
}

func (r *Runner) runTarget(ctx context.Context, t target.Target, workload target.Workload) (target.Result, error) {
	fmt.Printf("  setting up...\n")
	if err := t.Setup(ctx, workload); err != nil {
		return target.Result{}, fmt.Errorf("setup failed: %w", err)
	}

	defer func() {
		fmt.Printf("  tearing down...\n")
		t.Stop(ctx)
	}()

	fmt.Printf("  warming up (%d runs)...\n", r.config.WarmupRuns)
	for i := 0; i < r.config.WarmupRuns; i++ {
		if err := r.singleRun(ctx, t); err != nil {
			return target.Result{}, fmt.Errorf("warmup run %d failed: %w", i+1, err)
		}
	}

	fmt.Printf("  measuring (%d runs)...\n", r.config.Runs)
	var runResults []runResult
	for i := 0; i < r.config.Runs; i++ {
		result, err := r.measureRun(ctx, t)
		if err != nil {
			return target.Result{}, fmt.Errorf("measured run %d failed: %w", i+1, err)
		}
		runResults = append(runResults, result)
	}

	return aggregate(t.Name(), runResults), nil
}

func (r *Runner) singleRun(ctx context.Context, t target.Target) error {
	runCtx, cancel := context.WithTimeout(ctx, r.config.RunTimeout)
	defer cancel()

	if _, err := t.Start(runCtx); err != nil {
		return err
	}
	defer t.Stop(runCtx)

	_, err := runLoad(runCtx, t.Endpoint(), r.config.Load)
	return err
}

func (r *Runner) measureRun(ctx context.Context, t target.Target) (runResult, error) {
	runCtx, cancel := context.WithTimeout(ctx, r.config.RunTimeout)
	defer cancel()

	bootTime, err := t.Start(runCtx)
	if err != nil {
		return runResult{}, fmt.Errorf("start failed: %w", err)
	}
	defer t.Stop(runCtx)

	loadRes, err := runLoad(runCtx, t.Endpoint(), r.config.Load)
	if err != nil {
		return runResult{}, fmt.Errorf("load run failed: %w", err)
	}

	return runResult{
		bootTime:        bootTime,
		memoryIdle:      0,
		memoryUnderLoad: 0,
		throughput:      loadRes.Throughput,
		latencyP50:      loadRes.LatencyP50,
		latencyP95:      loadRes.LatencyP95,
		latencyP99:      loadRes.LatencyP99,
	}, nil
}

type runResult struct {
	bootTime        time.Duration
	memoryIdle      int64
	memoryUnderLoad int64
	throughput      float64
	latencyP50      float64
	latencyP95      float64
	latencyP99      float64
}

func aggregate(targetName string, runs []runResult) target.Result {
	return target.Result{
		Target:          targetName,
		BootTime:        medianDuration(bootTimes(runs)),
		MemoryIdle:      medianInt64(idleMemories(runs)),
		MemoryUnderLoad: medianInt64(loadMemories(runs)),
		Throughput:      medianFloat64(throughputs(runs)),
		LatencyP50:      medianFloat64(p50s(runs)),
		LatencyP95:      medianFloat64(p95s(runs)),
		LatencyP99:      medianFloat64(p99s(runs)),
		Metadata: target.Metadata{
			Runs: len(runs),
		},
	}
}
