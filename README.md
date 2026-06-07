# unimark

unimark benchmarks the same workload across containers, nano VMs, and unikernels — giving you an honest, reproducible picture of the tradeoffs.

    TARGET   BOOT TIME   THROUGHPUT    P50      P95      P99
    docker   116ms       31577 req/s   1.50ms   2.40ms   3.10ms
    nanos    3ms         44857 req/s   1.10ms   1.90ms   2.40ms

Most benchmarks tell you X is faster than Y. unimark tells you why, when, and under what conditions — so you can make an informed decision for your own workload.

## Background

Containers, nano VMs, and unikernels represent fundamentally different approaches to running software:

- **Docker** shares the host Linux kernel. Isolation is logical — namespaces and cgroups. Near-zero startup overhead.
- **Firecracker** runs a stripped-down Linux kernel inside a real VM. Strong isolation, AWS uses it in Lambda.
- **Nanos** fuses your application with a minimal custom kernel into a single image. No shell, no users, no other processes.

The tradeoffs between them depend on your workload, hardware, and constraints. unimark makes those tradeoffs visible.

## Prerequisites

- Go 1.22+ — https://golang.org/dl/
- Docker Desktop — https://www.docker.com/products/docker-desktop
- ops (NanoVMs CLI) — https://ops.city
- hey (load generator) — https://github.com/rakyll/hey

    curl https://ops.city/get.sh -sSfL | sh
    go install github.com/rakyll/hey@latest

## Install

    git clone https://github.com/shreyasbh/unimark
    cd unimark
    go build -o unimark .

## Run

    # default — Docker and Nanos, 5 runs, 30s load
    ./unimark run

    # quick single run
    ./unimark run --runs 1 --warmup 0 --duration 5s

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
| --output | table | table, json, or markdown |
| --file | stdout | Write output to file |

## Methodology

Results are the median of N runs, not the mean. A single slow run caused by a GC pause or scheduler hiccup does not skew results. Each result captures kernel version, CPU, and runtime versions so numbers are reproducible and comparable across machines.

See DESIGN.md for full methodology and architecture documentation.

## Roadmap

- Accurate Nanos boot time measurement
- Memory footprint metrics
- Firecracker target
- Attack surface metrics
- OSv and Unikraft targets

## Contributing

Adding a new target means implementing a single interface in pkg/target/. See DESIGN.md for details.

## License

Apache 2.0
