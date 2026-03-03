package orchestrator

import (
	"strings"
	"time"
)

type AggregateReport struct {
	TotalTasks     int
	Succeeded      int
	Failed         int
	Duration       time.Duration
	Results        []Result
	CombinedOutput string
}

type Aggregator struct{}

func (a Aggregator) Summarize(results []Result) *AggregateReport {
	report := &AggregateReport{
		TotalTasks: len(results),
		Results:    results,
	}

	var combined strings.Builder
	for _, result := range results {
		report.Duration += result.Duration

		combined.WriteString("--- Agent: ")
		combined.WriteString(result.Name)
		combined.WriteString(" ---\n")

		if result.Error != nil {
			report.Failed++
			combined.WriteString("[FAILED: ")
			combined.WriteString(result.Error.Error())
			combined.WriteString("]\n")
			continue
		}

		report.Succeeded++
		combined.WriteString(result.Output)
		combined.WriteByte('\n')
	}

	report.CombinedOutput = combined.String()
	return report
}
