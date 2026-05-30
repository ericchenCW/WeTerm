package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseFileDate(t *testing.T) {
	cases := map[string]struct {
		ok   bool
		year int
	}{
		"host-a-2026-05-21.jsonl": {true, 2026},
		"weird-name.jsonl":        {false, 0},
		"host-2026-13-01.jsonl":   {false, 0}, // bad month
		"x-2025-01-01.jsonl":      {true, 2025},
	}
	for name, want := range cases {
		got, ok := parseFileDate(name)
		if ok != want.ok {
			t.Errorf("%s: ok = %v, want %v", name, ok, want.ok)
			continue
		}
		if ok && got.Year() != want.year {
			t.Errorf("%s: year = %d, want %d", name, got.Year(), want.year)
		}
	}
}

func TestPruneRemovesOldFiles(t *testing.T) {
	dir := t.TempDir()

	now := time.Now()
	// Old file (15 days ago) — should be deleted.
	old := now.AddDate(0, 0, -15).Format("2006-01-02")
	// Recent file (2 days ago) — should be kept.
	recent := now.AddDate(0, 0, -2).Format("2006-01-02")

	oldPath := filepath.Join(dir, "host-"+old+".jsonl")
	recentPath := filepath.Join(dir, "host-"+recent+".jsonl")
	if err := os.WriteFile(oldPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(recentPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	code := runPrune([]string{"--data-dir", dir, "--keep-days", "7"})
	if code != 0 {
		t.Fatalf("runPrune exit code %d", code)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("old file should be deleted: %v", err)
	}
	if _, err := os.Stat(recentPath); err != nil {
		t.Errorf("recent file should be kept: %v", err)
	}
}

func TestPruneEmptyDir(t *testing.T) {
	dir := t.TempDir()
	code := runPrune([]string{"--data-dir", dir})
	if code != 0 {
		t.Errorf("empty dir should be ok, got code %d", code)
	}
}

func TestPruneMissingDir(t *testing.T) {
	code := runPrune([]string{"--data-dir", "/nonexistent/path/here"})
	if code != 0 {
		t.Errorf("missing dir should be ok, got code %d", code)
	}
}
