package main

import (
	"os"
	"path/filepath"
	"testing"
)

// buildFakeProc creates a temporary /proc-like tree for tests, sets procRoot
// to it, and returns a cleanup callback that restores the original root.
func buildFakeProc(t *testing.T, entries map[string]string) func() {
	t.Helper()
	dir := t.TempDir()
	for path, content := range entries {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	orig := procRoot
	procRoot = dir
	return func() { procRoot = orig }
}

func TestReadProcStat_SimpleComm(t *testing.T) {
	defer buildFakeProc(t, map[string]string{
		"1234/stat": "1234 (sleep) S 1 1234 1234 0 -1 0 0 0 0 0 100 200 0 0 20 0 1 0 100 0 0 0 0\n",
	})()
	s, err := readProcStat(1234)
	if err != nil {
		t.Fatal(err)
	}
	if s.comm != "sleep" {
		t.Errorf("comm = %q, want sleep", s.comm)
	}
	if s.utime != 100 || s.stime != 200 {
		t.Errorf("utime/stime = %d/%d, want 100/200", s.utime, s.stime)
	}
}

func TestReadProcStat_CommWithSpacesAndParens(t *testing.T) {
	// comm can contain spaces and unbalanced parens — must use LAST ')'.
	defer buildFakeProc(t, map[string]string{
		"99/stat": "99 (my (weird) comm) R 1 99 99 0 -1 0 0 0 0 0 7 11 0 0 20 0 1 0 100 0 0 0 0\n",
	})()
	s, err := readProcStat(99)
	if err != nil {
		t.Fatal(err)
	}
	if s.comm != "my (weird) comm" {
		t.Errorf("comm = %q", s.comm)
	}
	if s.utime != 7 || s.stime != 11 {
		t.Errorf("utime/stime = %d/%d, want 7/11", s.utime, s.stime)
	}
}

func TestReadVmRSS(t *testing.T) {
	defer buildFakeProc(t, map[string]string{
		"1/status": "Name:\tinit\nVmPeak:\t  12000 kB\nVmRSS:\t   4096 kB\nVmSize:\t  10000 kB\n",
	})()
	rss, err := readVmRSS(1)
	if err != nil {
		t.Fatal(err)
	}
	if rss != 4096 {
		t.Errorf("rss = %d, want 4096", rss)
	}
}

func TestReadVmRSS_KernelThread(t *testing.T) {
	// Kernel threads have no VmRSS line.
	defer buildFakeProc(t, map[string]string{
		"2/status": "Name:\tkthreadd\nState:\tS (sleeping)\n",
	})()
	rss, err := readVmRSS(2)
	if err != nil {
		t.Fatal(err)
	}
	if rss != 0 {
		t.Errorf("rss = %d, want 0 for kernel thread", rss)
	}
}

func TestReadProcIO(t *testing.T) {
	defer buildFakeProc(t, map[string]string{
		"77/io": "rchar: 100\nwchar: 200\nread_bytes: 4096\nwrite_bytes: 8192\n",
	})()
	io, err := readProcIO(77)
	if err != nil {
		t.Fatal(err)
	}
	if io == nil {
		t.Fatal("io == nil")
	}
	if io.readBytes != 4096 || io.writeBytes != 8192 {
		t.Errorf("read/write = %d/%d", io.readBytes, io.writeBytes)
	}
}

func TestReadProcCmdline(t *testing.T) {
	defer buildFakeProc(t, map[string]string{
		"55/cmdline": "java\x00-Xmx8g\x00-jar\x00order-svc.jar\x00",
	})()
	c, err := readProcCmdline(55)
	if err != nil {
		t.Fatal(err)
	}
	want := "java -Xmx8g -jar order-svc.jar"
	if c != want {
		t.Errorf("cmdline = %q, want %q", c, want)
	}
}

func TestReadProcCmdline_KernelThreadEmpty(t *testing.T) {
	defer buildFakeProc(t, map[string]string{
		"2/cmdline": "",
	})()
	c, err := readProcCmdline(2)
	if err != nil {
		t.Fatal(err)
	}
	if c != "" {
		t.Errorf("cmdline = %q, want empty", c)
	}
}

func TestReadSystemUptime(t *testing.T) {
	defer buildFakeProc(t, map[string]string{
		"uptime": "12345.67 89000.12\n",
	})()
	u, err := readSystemUptime()
	if err != nil {
		t.Fatal(err)
	}
	if u != 12345 {
		t.Errorf("uptime = %d, want 12345", u)
	}
}

func TestReadCwd(t *testing.T) {
	tmpdir := t.TempDir()
	target := filepath.Join(tmpdir, "srv/app-a")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	// Build a fake /proc tree where /proc/<pid>/cwd is a real symlink.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "100"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "100", "cwd")); err != nil {
		t.Fatal(err)
	}
	orig := procRoot
	procRoot = dir
	defer func() { procRoot = orig }()

	got, err := readCwd(100)
	if err != nil {
		t.Fatal(err)
	}
	if got != target {
		t.Errorf("readCwd = %q, want %q", got, target)
	}
}

func TestReadCwd_Missing(t *testing.T) {
	defer buildFakeProc(t, map[string]string{
		"100/.placeholder": "x",
	})()
	_, err := readCwd(100)
	if err == nil {
		t.Errorf("readCwd of missing symlink should error")
	}
}

func TestReadCgroup(t *testing.T) {
	defer buildFakeProc(t, map[string]string{
		"100/cgroup": "0::/system.slice/docker-abc.scope\n",
	})()
	got, err := readCgroup(100)
	if err != nil {
		t.Fatal(err)
	}
	if got != "0::/system.slice/docker-abc.scope\n" {
		t.Errorf("readCgroup payload mismatch: %q", got)
	}
}

func TestListPids(t *testing.T) {
	defer buildFakeProc(t, map[string]string{
		"1/stat":          "1 (init) S 0 0 0 0 -1 0 0 0 0 0 0 0 0 0 20 0 1 0 0 0 0 0 0\n",
		"123/stat":        "123 (sh) S 0 0 0 0 -1 0 0 0 0 0 0 0 0 0 20 0 1 0 0 0 0 0 0\n",
		"meminfo":         "MemTotal: 1\n", // non-numeric, should be skipped
		"self/should_skip": "x",            // "self" is a symlink in real /proc
	})()
	pids, err := listPids()
	if err != nil {
		t.Fatal(err)
	}
	got := map[int]bool{}
	for _, p := range pids {
		got[p] = true
	}
	if !got[1] || !got[123] {
		t.Errorf("missing expected pids: %v", pids)
	}
	if len(pids) != 2 {
		t.Errorf("unexpected pids: %v", pids)
	}
}
