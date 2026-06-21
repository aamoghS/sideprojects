package seo

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type BatchOptions struct {
	URLsFile  string
	Output    string
	ShardIdx  int
	ShardTotal int
}

func ParseBatchShard() (idx, total int, ok bool) {
	idxStr := strings.TrimSpace(os.Getenv("JOB_COMPLETION_INDEX"))
	totalStr := strings.TrimSpace(os.Getenv("JOB_COMPLETION_COUNT"))
	if idxStr == "" || totalStr == "" {
		return 0, 0, false
	}
	var err error
	idx, err = strconv.Atoi(idxStr)
	if err != nil {
		return 0, 0, false
	}
	total, err = strconv.Atoi(totalStr)
	if err != nil || total <= 0 || idx < 0 || idx >= total {
		return 0, 0, false
	}
	return idx, total, true
}

func LoadURLList(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var urls []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}
	return urls, sc.Err()
}

func ShardURLs(urls []string, idx, total int) []string {
	if total <= 1 {
		return urls
	}
	var out []string
	for i, u := range urls {
		if i%total == idx {
			out = append(out, u)
		}
	}
	return out
}

func RunBatch(opts BatchOptions) error {
	urls, err := LoadURLList(opts.URLsFile)
	if err != nil {
		return err
	}

	if idx, total, ok := ParseBatchShard(); ok {
		urls = ShardURLs(urls, idx, total)
	} else if opts.ShardTotal > 1 {
		urls = ShardURLs(urls, opts.ShardIdx, opts.ShardTotal)
	}

	out, err := os.Create(opts.Output)
	if err != nil {
		return err
	}
	defer out.Close()

	analyzer := NewAnalyzer()
	for _, raw := range urls {
		rec := AuditRecord{URL: raw}
		result, err := analyzer.Analyze(raw)
		if err != nil {
			rec.Error = err.Error()
		} else {
			rec.Score = result.Score
			rec.Title = result.Title
			rec.Issues = result.Issues
			rec.Fixes = result.Fixes
		}
		data, err := json.Marshal(rec)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, string(data)); err != nil {
			return err
		}
	}
	return nil
}
