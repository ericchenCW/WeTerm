package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var _ = strings.NewReader  // silence "unused" when refactoring

// TestCollect_EndToEnd builds a synthetic /proc, runs collect, and checks
// the resulting jsonl file contains records for each fake process.
func TestCollect_EndToEnd(t *testing.T) {
	defer buildFakeProc(t, map[string]string{
		// pid 1: regular process with io
		"1/stat":    "1 (init) S 0 1 1 0 -1 0 0 0 0 0 50 30 0 0 20 0 1 0 100 0 0 0 0\n",
		"1/status":  "Name:\tinit\nVmRSS:\t   1024 kB\n",
		"1/io":      "read_bytes: 1000\nwrite_bytes: 2000\n",
		"1/cmdline": "init\x00--default\x00",
		"1/cgroup":  "0::/init.scope\n", // not a container

		// pid 2: kernel thread
		"2/stat":    "2 (kthreadd) S 0 0 0 0 -1 0 0 0 0 0 1 2 0 0 20 0 1 0 0 0 0 0 0\n",
		"2/status":  "Name:\tkthreadd\n",
		"2/io":      "read_bytes: 0\nwrite_bytes: 0\n",
		"2/cmdline": "",
		"2/cgroup":  "0::/\n",

		// pid 123: java service
		"123/stat":    "123 (java) S 1 123 123 0 -1 0 0 0 0 0 5000 1000 0 0 20 0 1 0 100 0 0 0 0\n",
		"123/status":  "Name:\tjava\nVmRSS:\t  524288 kB\n",
		"123/io":      "read_bytes: 100000\nwrite_bytes: 200000\n",
		"123/cmdline": "java\x00-Xmx8g\x00-jar\x00order.jar\x00",
		"123/cgroup":  "0::/system.slice/myservice.service\n",

		"uptime": "1000.5 800.0\n",
	})()

	outDir := t.TempDir()
	code := runCollect([]string{"--data-dir", outDir})
	if code != 0 {
		t.Fatalf("runCollect exit code %d", code)
	}

	matches, err := filepath.Glob(filepath.Join(outDir, "*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 jsonl file, got %d", len(matches))
	}

	f, err := os.Open(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	pids := map[int]record{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var r record
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			t.Fatalf("bad jsonl: %v\nline: %s", err, scanner.Text())
		}
		pids[r.PID] = r
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}

	if len(pids) != 3 {
		t.Fatalf("got %d records, want 3: %v", len(pids), pids)
	}

	javaRec := pids[123]
	if javaRec.Comm != "java" {
		t.Errorf("pid 123 comm = %q", javaRec.Comm)
	}
	if javaRec.CPUJ != 6000 {
		t.Errorf("pid 123 cpu_j = %d, want 6000 (5000+1000)", javaRec.CPUJ)
	}
	if javaRec.RSSKB != 524288 {
		t.Errorf("pid 123 rss_kb = %d", javaRec.RSSKB)
	}
	if javaRec.IOR == nil || *javaRec.IOR != 100000 {
		t.Errorf("pid 123 io_r mismatch")
	}
	if !strings.Contains(javaRec.Cmdline, "order.jar") {
		t.Errorf("pid 123 cmdline = %q", javaRec.Cmdline)
	}
	if javaRec.UptimeS != 1000 {
		t.Errorf("pid 123 uptime = %d", javaRec.UptimeS)
	}
	if javaRec.Host == "" {
		t.Errorf("pid 123 host empty")
	}
	if javaRec.TS == 0 {
		t.Errorf("pid 123 ts == 0")
	}
}

// TestCollect_ProcessVanishedMidScan ensures that if a process appears in
// listPids() but its /proc/<pid>/stat is missing, we skip without erroring.
func TestCollect_ProcessVanishedMidScan(t *testing.T) {
	defer buildFakeProc(t, map[string]string{
		"1/stat":    "1 (init) S 0 1 1 0 -1 0 0 0 0 0 50 30 0 0 20 0 1 0 100 0 0 0 0\n",
		"1/status":  "Name:\tinit\nVmRSS:\t   1024 kB\n",
		"1/io":      "read_bytes: 0\nwrite_bytes: 0\n",
		"1/cmdline": "init\x00",
		"1/cgroup":  "0::/init.scope\n",
		// pid 99 exists as a directory entry (via listPids walking the tempdir
		// — we'll create the dir but no /stat inside) ...
		"99/.placeholder": "x",
		"uptime":          "100.0 90.0\n",
	})()

	outDir := t.TempDir()
	code := runCollect([]string{"--data-dir", outDir})
	if code != 0 {
		t.Fatalf("runCollect should not fail when a process vanishes; got %d", code)
	}
}

// TestCollect_ContainerProcess verifies cwd + container fields are populated
// when the process is in a Docker container.
func TestCollect_ContainerProcess(t *testing.T) {
	id := "1111222233334444555566667777888899990000aaaabbbbccccddddeeee0000"

	// Build fake /proc that includes a cwd symlink and a docker cgroup.
	dir := t.TempDir()
	pidDir := filepath.Join(dir, "555")
	if err := os.MkdirAll(pidDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cwdTarget := filepath.Join(dir, "_app")
	if err := os.MkdirAll(cwdTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(cwdTarget, filepath.Join(pidDir, "cwd")); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"stat":    "555 (python) S 1 555 555 0 -1 0 0 0 0 0 10 5 0 0 20 0 1 0 100 0 0 0 0\n",
		"status":  "Name:\tpython\nVmRSS:\t   2048 kB\n",
		"io":      "read_bytes: 1\nwrite_bytes: 2\n",
		"cmdline": "python\x00app.py\x00",
		"cgroup":  "0::/system.slice/docker-" + id + ".scope\n",
	} {
		if err := os.WriteFile(filepath.Join(pidDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(filepath.Join(dir, "uptime"), []byte("100 90\n"), 0o644)

	// Fake docker data-root with the real on-disk layout:
	//   <data-root>/containers/<id>/config.v2.json
	dataRoot := t.TempDir()
	containersDir := filepath.Join(dataRoot, "containers", id)
	os.MkdirAll(containersDir, 0o755)
	os.WriteFile(filepath.Join(containersDir, "config.v2.json"),
		[]byte(`{"Name":"/order-svc"}`), 0o644)

	origProc, origDocker := procRoot, dockerRoot
	procRoot = dir
	defer func() { procRoot = origProc; dockerRoot = origDocker }()

	outDir := t.TempDir()
	// Use the public CLI flag to point at our fake data-root — this also
	// exercises initDockerRoot's --docker-data-root override path.
	code := runCollect([]string{
		"--data-dir", outDir,
		"--docker-data-root", dataRoot,
	})
	if code != 0 {
		t.Fatalf("runCollect exit code %d", code)
	}

	matches, _ := filepath.Glob(filepath.Join(outDir, "*.jsonl"))
	if len(matches) != 1 {
		t.Fatalf("expected 1 file, got %d", len(matches))
	}
	data, _ := os.ReadFile(matches[0])
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var rec record
	for scanner.Scan() {
		var r record
		json.Unmarshal(scanner.Bytes(), &r)
		if r.PID == 555 {
			rec = r
		}
	}
	if rec.Cwd == nil || *rec.Cwd != cwdTarget {
		t.Errorf("Cwd = %v, want %q", rec.Cwd, cwdTarget)
	}
	if rec.Container == nil || *rec.Container != "order-svc" {
		t.Errorf("Container = %v, want order-svc", rec.Container)
	}
}
