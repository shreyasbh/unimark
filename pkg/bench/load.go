package bench

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type loadResult struct {
	Throughput float64
	LatencyP50 float64
	LatencyP95 float64
	LatencyP99 float64
}

func runLoad(ctx context.Context, endpoint string, config LoadConfig) (loadResult, error) {
	args := []string{
		"-n", strconv.Itoa(totalRequests(config)),
		"-c", strconv.Itoa(config.Connections),
		"-z", config.Duration.String(),
		endpoint + "/",
	}

	cmd := exec.CommandContext(ctx, "hey", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return loadResult{}, fmt.Errorf("hey failed: %w\n%s", err, output)
	}

	return parseHeyOutput(string(output))
}

func totalRequests(config LoadConfig) int {
	seconds := int(config.Duration.Seconds())
	return seconds * config.Connections * 1000
}

func parseHeyOutput(raw string) (loadResult, error) {
	var result loadResult

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)

		// Requests/sec: 13034.8499
		if strings.HasPrefix(line, "Requests/sec:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, err := strconv.ParseFloat(fields[1], 64)
				if err == nil {
					result.Throughput = val
				}
			}
		}

		// hey prints percentiles as "50%% in 0.0006 secs"
		// the %% is how Go's fmt prints a literal % — we handle both
		for _, pct := range []string{"50", "95", "99"} {
			prefix50 := pct + "%% in"
			prefix51 := pct + "% in"

			if strings.HasPrefix(line, prefix50) || strings.HasPrefix(line, prefix51) {
				fields := strings.Fields(line)
				// line is "50%% in 0.0006 secs" → fields[2] is the value
				if len(fields) >= 3 {
					val, err := strconv.ParseFloat(fields[2], 64)
					if err != nil {
						continue
					}
					ms := val * 1000
					switch pct {
					case "50":
						result.LatencyP50 = ms
					case "95":
						result.LatencyP95 = ms
					case "99":
						result.LatencyP99 = ms
					}
				}
			}
		}
	}

	if result.Throughput == 0 {
		return loadResult{}, fmt.Errorf("failed to parse throughput from hey output")
	}

	return result, nil
}
