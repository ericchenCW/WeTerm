package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

const defaultDataDir = "/var/log/procmon"

// record is the JSON Lines schema. Field order is stable across versions:
// ts, host, pid, comm, cmdline, cwd, container, cpu_j, rss_kb, io_r, io_w, uptime_s.
//
// Optional fields are pointers so they serialize as JSON null when absent,
// which is how the report-side loader detects "missing" vs "empty string".
type record struct {
	TS        int64   `json:"ts"`
	Host      string  `json:"host"`
	PID       int     `json:"pid"`
	Comm      string  `json:"comm"`
	Cmdline   string  `json:"cmdline"`
	Cwd       *string `json:"cwd"`
	Container *string `json:"container"`
	CPUJ      uint64  `json:"cpu_j"`
	RSSKB     uint64  `json:"rss_kb"`
	IOR       *uint64 `json:"io_r"`
	IOW       *uint64 `json:"io_w"`
	UptimeS   uint64  `json:"uptime_s"`
}

func runCollect(args []string) int {
	fs := flag.NewFlagSet("collect", flag.ExitOnError)
	dataDir := fs.String("data-dir", defaultDataDir, "output directory for jsonl files")
	dockerDataRoot := fs.String("docker-data-root", "",
		"override Docker data-root path (default: auto-detect from /etc/docker/daemon.json, fallback /var/lib/docker)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	initDockerRoot(*dockerDataRoot)

	host, err := os.Hostname()
	if err != nil {
		fmt.Fprintf(os.Stderr, "procmon: hostname error: %v\n", err)
		return 1
	}

	uptime, err := readSystemUptime()
	if err != nil {
		fmt.Fprintf(os.Stderr, "procmon: read uptime: %v\n", err)
		return 1
	}

	pids, err := listPids()
	if err != nil {
		fmt.Fprintf(os.Stderr, "procmon: list pids: %v\n", err)
		return 1
	}

	now := time.Now()
	ts := now.Unix()
	day := now.Format("2006-01-02")

	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "procmon: mkdir %s: %v\n", *dataDir, err)
		return 1
	}

	outPath := filepath.Join(*dataDir, fmt.Sprintf("%s-%s.jsonl", host, day))
	out, err := os.OpenFile(outPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "procmon: open %s: %v\n", outPath, err)
		return 1
	}
	defer out.Close()

	enc := json.NewEncoder(out)
	var (
		ok      int
		skipped int
	)

	for _, pid := range pids {
		rec, err := samplePid(pid)
		if err != nil {
			// Process-gone errors (ENOENT/ESRCH) are expected — count and move on.
			if isGone(err) {
				skipped++
				continue
			}
			fmt.Fprintf(os.Stderr, "procmon: pid %d: %v\n", pid, err)
			skipped++
			continue
		}
		rec.TS = ts
		rec.Host = host
		rec.UptimeS = uptime
		if err := enc.Encode(rec); err != nil {
			fmt.Fprintf(os.Stderr, "procmon: encode pid %d: %v\n", pid, err)
			skipped++
			continue
		}
		ok++
	}

	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "procmon: collected %d, skipped %d\n", ok, skipped)
	}
	return 0
}

func samplePid(pid int) (record, error) {
	stat, err := readProcStat(pid)
	if err != nil {
		return record{}, err
	}
	rss, err := readVmRSS(pid)
	if err != nil && !isGone(err) {
		// VmRSS missing for kernel threads — readVmRSS returns 0, nil for that.
		// Real errors (other than ENOENT) propagate.
		return record{}, err
	}
	io, err := readProcIO(pid)
	if err != nil && !isGone(err) {
		return record{}, err
	}
	cmdline, err := readProcCmdline(pid)
	if err != nil && !isGone(err) {
		return record{}, err
	}

	// cwd / container are best-effort optional fields. By the time we get
	// here, /proc/<pid>/stat already succeeded, so the process existed.
	// Any error reading these (vanished mid-call, permission denied, etc.)
	// just leaves the field null — we never propagate.
	cwd, _ := readCwd(pid)
	container, _ := resolveContainer(pid)

	rec := record{
		PID:     pid,
		Comm:    stat.comm,
		Cmdline: cmdline,
		CPUJ:    stat.utime + stat.stime,
		RSSKB:   rss,
	}
	if cwd != "" {
		rec.Cwd = &cwd
	}
	if container != "" {
		rec.Container = &container
	}
	if io != nil {
		r := io.readBytes
		w := io.writeBytes
		rec.IOR = &r
		rec.IOW = &w
	}
	return rec, nil
}

// isGone returns true if the error indicates the process disappeared
// between listing and reading its /proc files.
func isGone(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, fs.ErrNotExist)
}
