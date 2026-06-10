# unimark

unimark benchmarks the same workload across containers and unikernels — giving you an honest, reproducible picture of the tradeoffs.

    TARGET   MEM AVG    MEM PEAK   THROUGHPUT    P50      P95      P99
    docker   15.2 MiB   15.6 MiB   31k req/s     1.50ms   2.40ms   3.10ms
    nanos    71.6 MiB   75.8 MiB   46k req/s     1.00ms   1.90ms   2.30ms

Most benchmarks tell you X is faster than Y. unimark tells you why, when, and under what conditions — so you can make an informed decision for your own workload.

## Background

Containers and unikernels represent fundamentally different approaches to running software:

- **Docker** shares the host Linux kernel. Isolation is logical — namespaces and cgroups. Near-zero startup overhead.
- **Nanos** fuses your application with a minimal custom kernel into a single image. No shell, no users, no other processes. Launched via QEMU.
- **Firecracker** runs a stripped-down Linux kernel inside a real VM. AWS uses it in Lambda. Coming soon — requires Linux/KVM.

The tradeoffs depend on your workload. unimark makes them visible.

## Key findings

Running a simple Go HTTP server (IO bound workload):

- Nanos delivers 47% higher throughput — leaner network stack, fewer layers between app and NIC
- Nanos has lower latency across p50, p95, p99 — no Linux scheduler jitter
- Nanos uses 5x more memory — QEMU pre-allocates VM memory upfront
- Boot time is not a fair comparison — see --verbose for why

## Prerequisites

- Go 1.22+
- Docker Desktop — https://www.docker.com/products/docker-desktop
- ops (NanoVMs CLI) — https://ops.city
- hey (load generator)

    curl https://ops.city/get.sh -sSfL | sh
    go install github.com/rakyll/hey@latest

## Install

    git clone https://github.com/shreyasbh/unimark
    cd unimark
    go build -o unimark .

Build the workload binary for Nanos:

    cd workloads/go-http
    GOOS=linux GOARCH=arm64 go build -o server .  # Apple Silicon
    GOOS=linux GOARCH=amd64 go build -o server .  # Intel/AMD
    cd ../..

## Run

    # default — Docker and Nanos, 5 runs, 30s load
    ./unimark run

    # quick single run
    ./unimark run --runs 1 --warmup 0 --duration 5s

    # show boot phase breakdown
    ./unimark run --verbose

    # save results as markdown
    ./unimark run --output markdown --file results.md

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| --targets | docker,nanos | Comma separated list of targets |
| --runs | 5 | Measured runs per target |
| --warmup | 1 | Warmup runs to discard |
| --duration | 30s | Load generator duration |
| --connections | 50 | Concurrent connections |
| --verbose | false | Show boot phase breakdown |
| --output | table | table, json, or markdown |
| --file | stdout | Write output to file |

## Verbose mode

Boot time is not shown in the default table because Docker and Nanos boot times measure fundamentally different things — a process fork vs a full VM startup. Comparing them directly is misleading.

Use --verbose to see the phase breakdown:

    ./unimark run --verbose

    docker   total: 131ms
      container create    57ms
      network connect     54ms
      process + app       20ms

    nanos    total: 212ms
      qemu startup        208ms
      kernel boot         4ms
      app ready           2ms

The Nanos kernel boots in 4ms. QEMU takes 208ms. The fair comparison for boot time is unikernel vs Linux VM — which is why Firecracker is on the roadmap.

## Methodology

Results are the median of N runs, not the mean. Memory is sampled every 100ms during the load test — average and peak reported. Each result captures kernel version, CPU, and runtime versions for reproducibility.

See DESIGN.md for full methodology and measurement decisions.

## Roadmap

- CPU bound workload to show where unikernels lose their advantage
- Firecracker target (Linux/KVM required)
- Nested virtualization detection and warning
- Attack surface metrics
- OSv and Unikraft targets

## Contributing

Adding a new target means implementing a four-method interface in pkg/target/. See DESIGN.md for architecture details.

## License

Apache 2.0
