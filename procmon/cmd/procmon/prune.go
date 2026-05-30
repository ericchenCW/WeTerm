package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func runPrune(args []string) int {
	fs := flag.NewFlagSet("prune", flag.ExitOnError)
	dataDir := fs.String("data-dir", defaultDataDir, "directory to prune")
	keepDays := fs.Int("keep-days", 7, "delete files older than N days")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *keepDays < 1 {
		fmt.Fprintf(os.Stderr, "procmon: --keep-days must be >= 1\n")
		return 2
	}

	cutoff := time.Now().AddDate(0, 0, -*keepDays)
	entries, err := os.ReadDir(*dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0 // nothing to prune; not an error
		}
		fmt.Fprintf(os.Stderr, "procmon: readdir %s: %v\n", *dataDir, err)
		return 1
	}

	deleted := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		day, ok := parseFileDate(name)
		if !ok {
			continue // unrecognized name; leave alone
		}
		if day.Before(cutoff) {
			path := filepath.Join(*dataDir, name)
			if err := os.Remove(path); err != nil {
				fmt.Fprintf(os.Stderr, "procmon: remove %s: %v\n", path, err)
				continue
			}
			deleted++
		}
	}

	if deleted > 0 {
		fmt.Fprintf(os.Stderr, "procmon: pruned %d file(s)\n", deleted)
	}
	return 0
}

// parseFileDate extracts the YYYY-MM-DD date from a filename of the form
// "{host}-YYYY-MM-DD.jsonl". Returns false if the trailing 10-char date
// segment isn't a valid date.
func parseFileDate(name string) (time.Time, bool) {
	base := strings.TrimSuffix(name, ".jsonl")
	if len(base) < 10 {
		return time.Time{}, false
	}
	dateStr := base[len(base)-10:]
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
