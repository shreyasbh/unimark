package target

import (
	"context"
	"time"
)

type Target interface {
	Name() string
	Setup(ctx context.Context, workload Workload) error
	Start(ctx context.Context) (time.Duration, error)
	Stop(ctx context.Context) error
	Endpoint() string
}

type PhaseReporter interface {
	BootPhases() BootPhases
}

type Workload struct {
	Name string
	Port int
	Path string
	Env  map[string]string
}

type Result struct {
	Target      string
	BootTime    time.Duration
	BootPhases  *BootPhases
	MemoryAvg   int64
	MemoryPeak  int64
	Throughput  float64
	LatencyP50  float64
	LatencyP95  float64
	LatencyP99  float64
	Metadata    Metadata
}

type Metadata struct {
	Timestamp      time.Time
	KernelVersion  string
	CPU            string
	HostMemory     int64
	RuntimeVersion string
	Runs           int
}
