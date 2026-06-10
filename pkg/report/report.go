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

	fmt.Fprintln(w, "TARGET\tBOOT TIME\tMEM AVG\tMEM PEAK\tTHROUGHPUT\tP50\tP95\tP99")
	fmt.Fprintln(w, "------\t---------\t-------\t--------\t----------\t---\t---\t---")

	for _, res := range results {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%.0f req/s\t%.2fms\t%.2fms\t%.2fms\n",
			res.Target,
			res.BootTime.Round(time.Millisecond),
			formatBytes(res.MemoryAvg),
			formatBytes(res.MemoryPeak),
			res.Throughput,
			res.LatencyP50,
			res.LatencyP95,
			res.LatencyP99,
		)
	}

	if r.verbose {
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "BOOT PHASE BREAKDOWN")
		fmt.Fprintln(w, "--------------------")

		for _, res := range results {
			if res.BootPhases == nil {
				continue
			}
			fmt.Fprintf(w, "\n%s\n", res.Target)
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

			fmt.Fprintf(w, "  total             \t%s\n", p.Total.Round(time.Millisecond))
		}
	}

	return nil
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
