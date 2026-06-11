package report

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"github.com/sindhu/unimark/pkg/target"
)

type Renderer interface {
	Render(results []target.Result) error
}

type TableRenderer struct {
	out     io.Writer
	verbose bool
}

func NewTableRenderer(verbose bool) *TableRenderer {
	return &TableRenderer{out: os.Stdout, verbose: verbose}
}

func (r *TableRenderer) Render(results []target.Result) error {
	w := tabwriter.NewWriter(r.out, 0, 0, 3, ' ', 0)
	defer w.Flush()

	// Boot time and memory excluded from default table.
	// Boot time is environment dependent — QEMU on Mac vs bare metal KVM
	// produce very different numbers. Use --verbose for boot phase breakdown.
	// Memory is also environment dependent. Use --verbose for memory metrics.
	fmt.Fprintln(w, "TARGET\tTHROUGHPUT\tP50\tP95\tP99")
	fmt.Fprintln(w, "------\t----------\t---\t---\t---")

	for _, res := range results {
		fmt.Fprintf(w, "%s\t%.0f req/s\t%.2fms\t%.2fms\t%.2fms\n",
			res.Target,
			res.Throughput,
			res.LatencyP50,
			res.LatencyP95,
			res.LatencyP99,
		)
	}

	// recommendation line
	fmt.Fprintln(w, "")
	best, margin := recommendation(results)
	if best != "" {
		fmt.Fprintf(w, "%s shows %.0f%% higher throughput on this IO bound workload.\n", best, margin)
		fmt.Fprintf(w, "for CPU bound workloads run: unimark run --workload ./workloads/go-cpu\n")
	}

	if r.verbose {
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "MEMORY")
		fmt.Fprintln(w, "------")
		fmt.Fprintln(w, "Note: memory numbers are environment dependent.")
		fmt.Fprintln(w, "      QEMU on Mac pre-allocates VM memory differently than bare metal KVM.")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "TARGET\tMEM AVG\tMEM PEAK")
		fmt.Fprintln(w, "------\t-------\t--------")
		for _, res := range results {
			fmt.Fprintf(w, "%s\t%s\t%s\n",
				res.Target,
				formatBytes(res.MemoryAvg),
				formatBytes(res.MemoryPeak),
			)
		}

		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "BOOT PHASE BREAKDOWN")
		fmt.Fprintln(w, "--------------------")
		fmt.Fprintln(w, "Note: Docker boot time measures process fork.")
		fmt.Fprintln(w, "      Nanos boot time measures VM startup including QEMU.")
		fmt.Fprintln(w, "      These are not directly comparable.")
		fmt.Fprintln(w, "      Fair comparison: unikernel vs Linux VM (Firecracker coming soon).")
		fmt.Fprintln(w, "")

		for _, res := range results {
			if res.BootPhases == nil {
				continue
			}
			fmt.Fprintf(w, "\n%s   total: %s\n", res.Target, res.BootTime.Round(time.Millisecond))
			p := res.BootPhases

			switch res.Target {
			case "docker":
				if p.ContainerCreate > 0 {
					fmt.Fprintf(w, "  container create  \t%s\n", p.ContainerCreate.Round(time.Millisecond))
				}
				if p.NetworkConnect > 0 {
					fmt.Fprintf(w, "  network connect   \t%s\n", p.NetworkConnect.Round(time.Millisecond))
				}
				if p.ProcessStart > 0 {
					fmt.Fprintf(w, "  process + app     \t%s\n", p.ProcessStart.Round(time.Millisecond))
				}
			case "nanos":
				if p.QEMUStartup > 0 {
					fmt.Fprintf(w, "  qemu startup      \t%s\n", p.QEMUStartup.Round(time.Millisecond))
				}
				if p.KernelBoot > 0 {
					fmt.Fprintf(w, "  kernel boot       \t%s\n", p.KernelBoot.Round(time.Millisecond))
				}
				if p.AppReady > 0 {
					fmt.Fprintf(w, "  app ready         \t%s\n", p.AppReady.Round(time.Millisecond))
				}
			}
		}
	}

	return nil
}

// recommendation finds the best performing target and calculates margin
func recommendation(results []target.Result) (string, float64) {
	if len(results) < 2 {
		return "", 0
	}

	var best target.Result
	for _, r := range results {
		if r.Throughput > best.Throughput {
			best = r
		}
	}

	// find second best
	var second float64
	for _, r := range results {
		if r.Target != best.Target && r.Throughput > second {
			second = r.Throughput
		}
	}

	if second == 0 {
		return "", 0
	}

	margin := ((best.Throughput - second) / second) * 100
	return best.Target, margin
}

func formatBytes(b int64) string {
	if b == 0 {
		return "n/a"
	}
	switch {
	case b >= 1024*1024*1024:
		return fmt.Sprintf("%.1f GiB", float64(b)/1024/1024/1024)
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MiB", float64(b)/1024/1024)
	case b >= 1024:
		return fmt.Sprintf("%.1f KiB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
