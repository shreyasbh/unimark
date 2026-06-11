# unimark — Design Document

> An open source benchmarking harness for honest, reproducible comparisons across container, nano VM, and unikernel execution environments.

---

## 1. Goals

### Primary
- **Curiosity and learning** — understand how unikernels actually perform vs containers in practice, not just trust research papers
- **Community value** — build something the open source infrastructure community finds useful and can contribute to

### North star
A neutral, reproducible, community-maintained benchmarking tool that helps engineers choose the right execution environment for their workload — not by telling them which is faster, but by showing them where each excels and why.

---

## 2. Problem Statement

Research papers compare unikernels and containers but results are static, hardware-specific, and hard to reproduce. No neutral open source tool exists that lets engineers run the same benchmark on their own infrastructure and get comparable, structured results.

unimark fills that gap.

---

## 3. What unimark Measures

Three dimensions, implemented in phases:

| Dimension | MVP | v2 |
|---|---|---|
| Performance | ✅ | ✅ |
| Resource efficiency | ✅ | ✅ |
| Attack surface | — | ✅ |

### Performance metrics
- **Throughput** — requests per second at a given concurrency level
- **Latency distribution** — p50, p95, p99 response times under load

### Resource efficiency metrics
- **Memory average** — average memory consumed during the load test
- **Memory peak** — maximum memory consumed during the load test

### Boot time
Boot time is captured but not shown in the default table. See section 14 for why.

### Attack surface metrics (v2)
- Number of syscalls available
- Number of running processes
- Open ports beyond the workload
- Binary and library count in the image

---

## 4. Targets

Three execution environments, each representing a distinct architectural approach:

| Target | Type | Isolation | Kernel |
|---|---|---|---|
| Docker | Container | Namespace + cgroup | Shared host Linux |
| Firecracker | Nano VM | Full VM boundary | Minimal Linux (guest) |
| Nanos | Unikernel | Full VM boundary | Custom (non-Linux) |

### Why these three
- **Docker** — the baseline. Shared kernel, namespace isolation, near-zero startup overhead
- **Firecracker** — AWS-built, open source (Apache 2.0), production-proven in Lambda. Represents the nano VM category
- **Nanos** — open source (Apache 2.0), NanoVMs. Most accessible unikernel via the `ops` CLI. Represents the unikernel category

### Key architectural differences

**Docker** runs as a Linux process with namespace isolation. The host kernel does all the work. Startup is a process fork.

**Firecracker** launches a real VM using KVM. Runs a stripped-down Linux kernel inside. Configured via REST API over a Unix socket. Startup involves hypervisor initialization and kernel boot.

**Nanos** packages your application binary with the Nanos kernel into a single image. Launched via QEMU/KVM. No shell, no users, no other processes. Application and kernel are fused into one artifact.

---

## 5. Workload

### IO bound — go-http
A simple Go HTTP server returning a static response. Spends most time on network I/O. This is where Nanos's leaner network stack shows an advantage.

### CPU bound — go-cpu (coming soon)
A Go HTTP server that computes fibonacci for each request. Spends most time on CPU. Expected to show Docker and Nanos performing equally — the OS overhead is irrelevant when the bottleneck is computation.

The two workloads together tell the complete story — unikernels win on IO bound, lose their advantage on CPU bound.

**Why Go**
- Nanos has excellent Go support
- Firecracker runs any Linux binary including Go
- Most credible benchmarks in this space use Go, making results comparable
- unimark itself is written in Go — same language end to end

---

## 6. Methodology

### Benchmark lifecycle per target
```
Setup → Warmup runs (discarded) → Measured runs → Aggregate → Teardown
```

### Memory measurement
Sampled every 100ms during the full load test duration. Average and peak reported. A single snapshot is noisy — memory allocators release and acquire memory continuously. Continuous sampling gives a more honest picture.

### Load generation
Uses `hey` — a simple HTTP load generator written in Go. Duration-based runs (default 30s).

### Aggregation
Minimum 5 measured runs per target. Results aggregated using **median**, not mean. A single slow run caused by a GC pause or scheduler hiccup will not skew results.

### Environment capture
Every result captures host state for reproducibility:
- Kernel version
- CPU model
- Total host memory
- Runtime version (Docker version, ops version, Firecracker version)
- Timestamp
- Number of runs aggregated

Results without environment context are not reproducible and not trustworthy.

### Warmup
At least 1 warmup run per target, discarded before measurement. Ensures JIT compilation, filesystem caches, and network setup are stable before measuring.

---

## 7. Architecture

### Design principles
- **Plugin architecture** — targets, metric collectors, and report renderers are all behind interfaces. New targets and output formats can be added without touching core logic.
- **Context-aware** — every operation accepts `context.Context`. Cancellation and timeouts propagate cleanly through the entire call chain.
- **Measure from outside** — the load generator is the source of truth, not self-reported metrics.
- **Reproducibility first** — environment state is captured with every result.

### Core interface

```go
type Target interface {
    Name() string
    Setup(ctx context.Context, workload Workload) error
    Start(ctx context.Context) (time.Duration, error)
    Stop(ctx context.Context) error
    Endpoint() string
}
```

### Optional interfaces

```go
// PhaseReporter is implemented by targets that can report boot phase breakdown
type PhaseReporter interface {
    BootPhases() BootPhases
}

// memoryReader is implemented by targets that report their own memory
type memoryReader interface {
    Memory(ctx context.Context) (int64, error)
}
```

These are separate from the core interface so they are optional. The runner uses type assertions to check if a target supports them.

### Project structure

```
unimark/
├── main.go
├── cmd/
│   ├── root.go              # Cobra CLI entry point
│   └── run.go               # unimark run command
├── pkg/
│   ├── target/
│   │   ├── target.go        # Target interface, Workload, Result, Metadata
│   │   ├── phases.go        # BootPhases struct
│   │   ├── docker.go        # Docker implementation
│   │   ├── firecracker.go   # Firecracker implementation (coming soon)
│   │   └── nanos.go         # Nanos implementation
│   ├── bench/
│   │   ├── runner.go        # Orchestrates setup, warmup, run, teardown
│   │   ├── load.go          # Load generator (hey)
│   │   └── stats.go         # Median aggregation helpers
│   ├── metrics/
│   │   └── metrics.go       # Memory and environment capture
│   └── report/
│       ├── report.go        # Renderer interface + table renderer
│       ├── json.go          # JSON renderer
│       └── markdown.go      # Markdown renderer
├── workloads/
│   ├── go-http/             # IO bound workload
│   │   ├── main.go
│   │   └── Dockerfile
│   └── go-cpu/              # CPU bound workload (coming soon)
│       ├── main.go
│       └── Dockerfile
├── DESIGN.md
└── README.md
```

---

## 8. CLI

```bash
# run all targets with defaults
unimark run --workload ./workloads/go-http

# run specific targets
unimark run --targets docker,nanos

# show boot phase breakdown
unimark run --targets docker,nanos --verbose

# 10 measured runs, output as markdown
unimark run --runs 10 --output markdown --file results.md

# custom load
unimark run --connections 100 --duration 60s
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--workload` | `./workloads/go-http` | Path to workload |
| `--targets` | `docker,nanos` | Targets to benchmark |
| `--runs` | `5` | Measured runs per target |
| `--warmup` | `1` | Warmup runs to discard |
| `--port` | `8080` | Port the workload listens on |
| `--timeout` | `5m` | Timeout per run |
| `--connections` | `50` | Concurrent connections |
| `--duration` | `30s` | Load generator duration |
| `--verbose` | false | Show boot phase breakdown |
| `--output` | `table` | Output format: table, json, markdown |
| `--file` | stdout | Write output to file |

---

## 9. Output

### Default table
```
host: darwin | arch: arm64 | kernel: 25.4.0

TARGET   MEM AVG    MEM PEAK   THROUGHPUT    P50      P95      P99
------   -------    --------   ----------    ---      ---      ---
docker   15.2 MiB   15.6 MiB   31k req/s     1.50ms   2.40ms   3.10ms
nanos    71.6 MiB   75.8 MiB   46k req/s     1.00ms   1.90ms   2.30ms
```

### Verbose table (--verbose)
Adds boot phase breakdown below the main table with a note explaining why boot time is not directly comparable across targets.

### JSON
Structured output for tooling and further analysis.

### Markdown
GitHub-ready table with environment metadata. Designed for sharing in issues, READMEs, and blog posts.

---

## 10. Implementation Language

**Go** — for the following reasons:

- Single binary distribution, no runtime dependencies
- Strong subprocess management for orchestrating Docker, QEMU, ops CLI
- Cobra CLI framework — same used by kubectl, Hugo
- ops CLI is written in Go — potential for upstream contribution
- Target audience is infrastructure engineers who expect Go tooling
- Workload is also Go — same language end to end

---

## 11. Dependencies

### Runtime
- `github.com/spf13/cobra` — CLI framework

### External tools (must be installed)
- `docker` — Docker CLI
- `ops` — NanoVMs CLI for Nanos
- `hey` — HTTP load generator (`go install github.com/rakyll/hey@latest`)
- `firecracker` — Firecracker VMM (Linux only, coming soon)

---

## 12. Roadmap

### MVP (v0.1) — current
- Docker and Nanos targets
- Go HTTP workload (IO bound)
- Memory avg/peak, throughput, latency metrics
- Boot phase breakdown via --verbose
- Table, JSON, markdown output
- Environment capture

### v0.2
- CPU bound workload (go-cpu)
- Firecracker target (Linux/KVM required)
- Nested virtualization detection and warning

### v0.3
- Attack surface metrics
- Concurrency sweep — test at multiple connection levels automatically
- CI-friendly output

### Future
- Additional unikernels (OSv, Unikraft)
- Web UI for result visualization
- Community result sharing

---

## 13. Contributing

unimark welcomes contributions. The plugin architecture is designed specifically to make adding new targets and renderers straightforward.

**Adding a new target** — implement the `Target` interface in `pkg/target/`, register it in `cmd/run.go`.

**Adding a new renderer** — implement the `Renderer` interface in `pkg/report/`.

**Adding a new workload** — add a directory under `workloads/` with a Dockerfile and ops-compatible binary.

---

## 14. Measurement Decisions

This section documents deliberate choices about what to measure, what not to measure, and why.

### Boot time not shown by default

Boot time is excluded from the default output table. It is available via `--verbose`.

**Why:**

Docker boot time measures the time to fork a process and set up Linux namespaces and cgroups. There is no kernel boot involved — the host kernel is already running and shared.

Nanos boot time measures the time to start a QEMU hypervisor, initialize virtual hardware, boot the Nanos kernel, and start the application. This includes ~200ms of QEMU startup that has nothing to do with Nanos or the application itself.

Showing these two numbers side by side implies they are measuring the same thing. They are not.

The fair comparison for boot time is unikernel vs Linux VM — both boot a kernel, both pay a hypervisor cost. That comparison is on the unimark roadmap when Firecracker support is added.

**What --verbose shows instead:**

A phase breakdown that explains where the time goes:

```
docker   total: 131ms
  container create    57ms    namespace + cgroup setup
  network connect     54ms    virtual NIC + bridge
  process + app       20ms    fork + app startup

nanos    total: 212ms
  qemu startup        208ms   hypervisor + virtual hardware init
  kernel boot         4ms     nanos kernel initialization
  app ready           2ms     application startup
```

This tells the honest story — Nanos kernel and app start in milliseconds, but QEMU dominates total time.

### Memory measured as average and peak during load

Memory is sampled every 100ms during the load test rather than at a single point in time. A single snapshot is noisy — memory allocators release and acquire memory continuously. Average and peak across the full load test duration gives a more honest picture.

### Throughput and latency are the primary metrics

These are the metrics that matter most for the Docker vs Nanos comparison because:
- Both run the same workload
- Both are measured by the same external load generator
- The comparison is fair — same requests, same concurrency, same duration
- The results reflect real architectural differences — Nanos network stack vs Linux network stack

### Nested virtualization warning (v0.2)

Running Nanos inside a VM (e.g. a cloud instance) introduces nested virtualization — QEMU running inside another hypervisor. This significantly degrades Nanos performance. unimark will detect and warn about this in v0.2.

---

*unimark is a personal exploration project driven by curiosity about unikernel performance and a desire to build something genuinely useful for the infrastructure community.*
