package report

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sindhu/unimark/pkg/target"
)

type MarkdownRenderer struct {
	outputPath string
}

func NewMarkdownRenderer(outputPath string) *MarkdownRenderer {
	return &MarkdownRenderer{outputPath: outputPath}
}

func (r *MarkdownRenderer) Render(results []target.Result) error {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# unimark results\n\n")
	fmt.Fprintf(&sb, "generated: %s\n\n", time.Now().Format(time.RFC3339))

	if len(results) > 0 {
		meta := results[0].Metadata
		fmt.Fprintf(&sb, "## environment\n\n")
		fmt.Fprintf(&sb, "- kernel: %s\n", meta.KernelVersion)
		fmt.Fprintf(&sb, "- cpu: %s\n", meta.CPU)
		fmt.Fprintf(&sb, "- memory: %s\n", formatBytes(meta.HostMemory))
		fmt.Fprintf(&sb, "- runs per target: %d\n\n", meta.Runs)
	}

	fmt.Fprintf(&sb, "## results\n\n")
	fmt.Fprintf(&sb, "| target | boot time | throughput | p50 | p95 | p99 |\n")
	fmt.Fprintf(&sb, "|--------|-----------|------------|-----|-----|-----|\n")

	for _, res := range results {
		fmt.Fprintf(&sb, "| %s | %s | %.0f req/s | %.2fms | %.2fms | %.2fms |\n",
			res.Target,
			res.BootTime.Round(time.Millisecond),
			res.Throughput,
			res.LatencyP50,
			res.LatencyP95,
			res.LatencyP99,
		)
	}

	if r.outputPath == "" {
		fmt.Println(sb.String())
		return nil
	}

	if err := os.WriteFile(r.outputPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("failed to write markdown: %w", err)
	}

	fmt.Printf("results written to %s\n", r.outputPath)
	return nil
}
