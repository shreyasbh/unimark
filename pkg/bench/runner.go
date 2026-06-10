package bench

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sindhu/unimark/pkg/metrics"
	"github.com/sindhu/unimark/pkg/target"
)

type Config struct {
	Runs       int
	WarmupRuns int
	RunTimeout time.Duration
	Verbose    bool
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
		fmt.Printf("→ benchmarking %s\n", t.Name())

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

	return aggregate(t.Name(), runResults, t), nil
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

type memoryReader interface {
	Memory(ctx context.Context) (int64, error)
}

type pidGetter interface {
	PID() (int, error)
}

func (r *Runner) measureRun(ctx context.Context, t target.Target) (runResult, error) {
	runCtx, cancel := context.WithTimeout(ctx, r.config.RunTimeout)
	defer cancel()

	bootTime, err := t.Start(runCtx)
	if err != nil {
		return runResult{}, fmt.Errorf("start failed: %w", err)
	}
	defer t.Stop(runCtx)

	// sample memory continuously during load test
	var samples []int64
	var sampleMu sync.Mutex
	samplingDone := make(chan struct{})

	go func() {
		for {
			select {
			case <-samplingDone:
				return
			default:
				mem := r.measureMemory(runCtx, t)
				if mem > 0 {
					sampleMu.Lock()
					samples = append(samples, mem)
					sampleMu.Unlock()
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	// run load
	loadRes, err := runLoad(runCtx, t.Endpoint(), r.config.Load)

	// stop memory sampling
	close(samplingDone)

	if err != nil {
		return runResult{}, fmt.Errorf("load run failed: %w", err)
	}

	// calculate avg and peak from samples
	sampleMu.Lock()
	memAvg, memPeak := calcMemoryStats(samples)
	sampleMu.Unlock()

	return runResult{
		bootTime:   bootTime,
		memoryAvg:  memAvg,
		memoryPeak: memPeak,
		throughput: loadRes.Throughput,
		latencyP50: loadRes.LatencyP50,
		latencyP95: loadRes.LatencyP95,
		latencyP99: loadRes.LatencyP99,
	}, nil
}

// calcMemoryStats returns average and peak from a slice of memory samples
func calcMemoryStats(samples []int64) (avg int64, peak int64) {
	if len(samples) == 0 {
		return 0, 0
	}

	var sum int64
	for _, s := range samples {
		sum += s
		if s > peak {
			peak = s
		}
	}
	avg = sum / int64(len(samples))
	return
}

func (r *Runner) measureMemory(ctx context.Context, t target.Target) int64 {
	if mr, ok := t.(memoryReader); ok {
		mem, err := mr.Memory(ctx)
		if err == nil {
			return mem
		}
	}

	if pg, ok := t.(pidGetter); ok {
		pid, err := pg.PID()
		if err == nil {
			mem, err := metrics.ReadProcessMemory(pid)
			if err == nil {
				return mem
			}
		}
	}

	return 0
}

type runResult struct {
	bootTime   time.Duration
	memoryAvg  int64
	memoryPeak int64
	throughput float64
	latencyP50 float64
	latencyP95 float64
	latencyP99 float64
}

func aggregate(targetName string, runs []runResult, t target.Target) target.Result {
	result := target.Result{
		Target:     targetName,
		BootTime:   medianDuration(bootTimes(runs)),
		MemoryAvg:  medianInt64(avgMemories(runs)),
		MemoryPeak: medianInt64(peakMemories(runs)),
		Throughput: medianFloat64(throughputs(runs)),
		LatencyP50: medianFloat64(p50s(runs)),
		LatencyP95: medianFloat64(p95s(runs)),
		LatencyP99: medianFloat64(p99s(runs)),
		Metadata: target.Metadata{
			Runs: len(runs),
		},
	}

	if pr, ok := t.(target.PhaseReporter); ok {
		phases := pr.BootPhases()
		result.BootPhases = &phases
	}

	return result
}
