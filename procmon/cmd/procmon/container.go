package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"regexp"
)

// dockerRoot is the path where Docker stores per-container metadata —
// always `<data-root>/containers`. Production code calls initDockerRoot
// once at process start to auto-detect from /etc/docker/daemon.json;
// tests assign it directly.
var dockerRoot = "/var/lib/docker/containers"

// initDockerRoot configures dockerRoot from (in priority order):
//   1. an explicit override (CLI --docker-data-root, set via the override arg)
//   2. /etc/docker/daemon.json's `data-root` field
//   3. the default /var/lib/docker
//
// Idempotent: callers in `runCollect` invoke it once per process. The
// daemon.json read is a few hundred bytes — sub-millisecond, fine to do
// on every collect tick.
func initDockerRoot(override string) {
	if override != "" {
		dockerRoot = override + "/containers"
		return
	}
	if root := readDaemonJSONDataRoot("/etc/docker/daemon.json"); root != "" {
		dockerRoot = root + "/containers"
		return
	}
	dockerRoot = "/var/lib/docker/containers"
}

// readDaemonJSONDataRoot returns the `data-root` field from a Docker
// daemon.json file, or "" if the file is missing, unreadable, malformed,
// or has no data-root set. Returns "" on any error — caller falls back
// to the standard default.
func readDaemonJSONDataRoot(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var cfg struct {
		DataRoot string `json:"data-root"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return ""
	}
	return cfg.DataRoot
}

// 64-hex container ID — Docker, containerd, cri-containerd, podman all use
// this for the canonical ID. We deliberately keep the regex anchored on the
// surrounding separators commonly found in cgroup paths so that we don't
// match accidental hex strings (e.g. pod UUIDs).
// All regexes use `(?m)` so `$` matches end-of-line (cgroup files are
// newline-terminated). The trailing alternation lets us accept the ID
// followed by `.scope`, `/<subpath>`, or end-of-line — all real shapes
// observed across Docker/containerd/podman / cgroup v1 vs v2.
var (
	// Matches:
	//   /docker/<64hex>              (Docker cgroupv1, ID followed by \n or /)
	//   docker-<64hex>.scope         (Docker cgroupv2, systemd-managed)
	dockerIDRE = regexp.MustCompile(`(?m)(?:/|^)docker[-/]([0-9a-f]{64})(?:\.scope|/|$)`)

	// Matches:
	//   cri-containerd-<64hex>.scope (Kubernetes via cri-containerd)
	//   containerd-<64hex>.scope     (containerd standalone)
	containerdIDRE = regexp.MustCompile(`(?:cri-)?containerd-([0-9a-f]{64})\.scope`)

	// Matches:
	//   libpod-<64hex>.scope         (Podman)
	//   podman-<64hex>.scope         (older Podman naming)
	podmanIDRE = regexp.MustCompile(`(?:libpod|podman)-([0-9a-f]{64})\.scope`)
)

// extractContainerID scans a /proc/<pid>/cgroup payload and returns the
// 64-hex container ID if any known pattern matches. Returns "" for
// non-container processes (systemd units, plain processes, kernel threads).
func extractContainerID(cgroup string) string {
	for _, re := range []*regexp.Regexp{dockerIDRE, containerdIDRE, podmanIDRE} {
		if m := re.FindStringSubmatch(cgroup); len(m) >= 2 {
			return m[1]
		}
	}
	return ""
}

// dockerName reads Docker's per-container metadata file and returns the
// configured container name (without the leading '/'). Returns ("", false)
// if the file is missing (non-Docker or custom data-root), unreadable, or
// malformed — caller is expected to fall back to short ID in that case.
//
// We deliberately read & decode only the Name field, not the entire
// (sometimes huge) config blob.
func dockerName(containerID string) (string, bool) {
	path := dockerRoot + "/" + containerID + "/config.v2.json"
	data, err := os.ReadFile(path)
	if err != nil {
		// Any error here (missing file, permission, etc.) → caller falls
		// back. Don't propagate — it's an expected miss for non-Docker hosts.
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission) {
			return "", false
		}
		return "", false
	}
	var meta struct {
		Name string `json:"Name"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return "", false
	}
	if meta.Name == "" {
		return "", false
	}
	// Docker stores names as "/order-svc" — strip the leading slash.
	if meta.Name[0] == '/' {
		return meta.Name[1:], true
	}
	return meta.Name, true
}

// resolveContainer returns the best-effort container identifier for a pid,
// or "" if the process is not in a container. Priority:
//
//  1. Docker container with resolvable name  → "<name>"
//  2. Any container ID extractable           → 12-char short ID
//  3. No container pattern matched           → ""
func resolveContainer(pid int) (string, error) {
	cg, err := readCgroup(pid)
	if err != nil {
		return "", err
	}
	id := extractContainerID(cg)
	if id == "" {
		return "", nil
	}
	if name, ok := dockerName(id); ok {
		return name, nil
	}
	// Short ID — Docker convention is the first 12 hex characters.
	return id[:12], nil
}
