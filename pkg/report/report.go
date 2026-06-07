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
	out io.Writer
}

func NewTableRenderer() *TableRenderer {
	return &TableRenderer{out: os.Stdout}
}

func (r *TableRenderer) Render(results []target.Result) error {
	w := tabwriter.NewWriter(r.out, 0, 0, 3, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "TARGET\tBOOT TIME\tTHROUGHPUT\tP50\tP95\tP99")
	fmt.Fprintln(w, "------\t---------\t----------\t---\t---\t---")

	for _, res := range results {
		fmt.Fprintf(w, "%s\t%s\t%.0f req/s\t%.2fms\t%.2fms\t%.2fms\n",
			res.Target,
			res.BootTime.Round(time.Millisecond),
			res.Throughput,
			res.LatencyP50,
			res.LatencyP95,
			res.LatencyP99,
		)
	}

	return nil
}

func formatBytes(b int64) string {
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
