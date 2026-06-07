package bench

import (
	"sort"
	"time"
)

func medianDuration(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(values))
	copy(sorted, values)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	return sorted[len(sorted)/2]
}

func medianFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	return sorted[len(sorted)/2]
}

func medianInt64(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]int64, len(values))
	copy(sorted, values)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	return sorted[len(sorted)/2]
}

func bootTimes(runs []runResult) []time.Duration {
	out := make([]time.Duration, len(runs))
	for i, r := range runs {
		out[i] = r.bootTime
	}
	return out
}

func idleMemories(runs []runResult) []int64 {
	out := make([]int64, len(runs))
	for i, r := range runs {
		out[i] = r.memoryIdle
	}
	return out
}

func loadMemories(runs []runResult) []int64 {
	out := make([]int64, len(runs))
	for i, r := range runs {
		out[i] = r.memoryUnderLoad
	}
	return out
}

func throughputs(runs []runResult) []float64 {
	out := make([]float64, len(runs))
	for i, r := range runs {
		out[i] = r.throughput
	}
	return out
}

func p50s(runs []runResult) []float64 {
	out := make([]float64, len(runs))
	for i, r := range runs {
		out[i] = r.latencyP50
	}
	return out
}

func p95s(runs []runResult) []float64 {
	out := make([]float64, len(runs))
	for i, r := range runs {
		out[i] = r.latencyP95
	}
	return out
}

func p99s(runs []runResult) []float64 {
	out := make([]float64, len(runs))
	for i, r := range runs {
		out[i] = r.latencyP99
	}
	return out
}
