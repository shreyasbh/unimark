package report

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sindhu/unimark/pkg/target"
)

type JSONRenderer struct {
	outputPath string
}

func NewJSONRenderer(outputPath string) *JSONRenderer {
	return &JSONRenderer{outputPath: outputPath}
}

func (r *JSONRenderer) Render(results []target.Result) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if r.outputPath == "" {
		fmt.Println(string(data))
		return nil
	}

	if err := os.WriteFile(r.outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write results: %w", err)
	}

	fmt.Printf("results written to %s\n", r.outputPath)
	return nil
}
