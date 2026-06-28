package app

import (
	"encoding/json"
	"os"
	"time"

	"github.com/aamoghS/sideprojects/movie/internal/agent"
)

type jsonMovie struct {
	Title  string `json:"title"`
	Year   int    `json:"year"`
	Plot   string `json:"plot"`
	Score  int    `json:"score,omitempty"`
	Source string `json:"source"`
}

type jsonAgentResult struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Proxy  string      `json:"proxy"`
	Error  string      `json:"error,omitempty"`
	Movies []jsonMovie `json:"movies"`
}

type jsonOutput struct {
	Elapsed string            `json:"elapsed"`
	Results []jsonAgentResult `json:"results"`
}

func writeJSONResults(path string, results []agent.Result, start time.Time) error {
	out := jsonOutput{
		Elapsed: time.Since(start).Round(time.Millisecond).String(),
		Results: make([]jsonAgentResult, 0, len(results)),
	}
	for _, r := range results {
		jr := jsonAgentResult{
			ID:    r.Agent.ID,
			Name:  r.Agent.Name,
			Proxy: r.Proxy,
			Error: r.Error,
		}
		for _, m := range r.Movies {
			jr.Movies = append(jr.Movies, jsonMovie{
				Title:  m.Title,
				Year:   m.Year,
				Plot:   m.Plot,
				Score:  m.Score,
				Source: m.Source,
			})
		}
		out.Results = append(out.Results, jr)
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
