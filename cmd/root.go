package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "unimark",
	Short: "Benchmark Docker, Firecracker, and Nanos on the same workload",
	Long: `unimark is an open source benchmarking harness that gives engineers
an honest, reproducible comparison across container, nano VM, and unikernel
execution environments.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("output", "o", "", "output format: table, json, markdown")
	rootCmd.PersistentFlags().StringP("file", "f", "", "write output to file instead of stdout")
}
