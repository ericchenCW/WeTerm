package main

import (
	"bufio"
	"bytes"
	"errors"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

// procRoot is overridable in tests; production uses /proc.
var procRoot = "/proc"

// procStat holds fields extracted from /proc/<pid>/stat.
type procStat struct {
	comm  string
	utime uint64
	stime uint64
}

// readProcStat parses /proc/<pid>/stat. The comm field is in parentheses
// and may contain spaces and unbalanced parens — the standard trick is to
// locate the LAST ')' in the line, then split the remainder by spaces.
//
// /proc/<pid>/stat format (selected, 1-based):
//
//	1  pid
//	2  comm
//	3  state
//	...
//	14 utime
//	15 stime
func readProcStat(pid int) (procStat, error) {
	data, err := os.ReadFile(procRoot + "/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return procStat{}, err
	}

	rparen := bytes.LastIndexByte(data, ')')
	if rparen < 0 {
		return procStat{}, errors.New("malformed stat: no ')'")
	}
	lparen := bytes.IndexByte(data, '(')
	if lparen < 0 || lparen >= rparen {
		return procStat{}, errors.New("malformed stat: no '('")
	}

	comm := string(data[lparen+1 : rparen])

	rest := strings.Fields(string(data[rparen+1:]))
	// rest[0] = state, rest[1] = ppid, ..., utime is field 14 overall,
	// so rest index = 14 - 3 = 11; stime = 12.
	if len(rest) < 13 {
		return procStat{}, errors.New("malformed stat: too few fields")
	}

	utime, err := strconv.ParseUint(rest[11], 10, 64)
	if err != nil {
		return procStat{}, err
	}
	stime, err := strconv.ParseUint(rest[12], 10, 64)
	if err != nil {
		return procStat{}, err
	}

	return procStat{comm: comm, utime: utime, stime: stime}, nil
}

// readVmRSS parses /proc/<pid>/status for VmRSS in KB. Kernel threads
// have no VmRSS line — return 0 in that case (not an error).
func readVmRSS(pid int) (uint64, error) {
	f, err := os.Open(procRoot + "/" + strconv.Itoa(pid) + "/status")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				return 0, errors.New("malformed VmRSS line")
			}
			return strconv.ParseUint(fields[1], 10, 64)
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, nil
}

// procIO holds fields from /proc/<pid>/io.
type procIO struct {
	readBytes  uint64
	writeBytes uint64
}

// readProcIO parses /proc/<pid>/io. Returns (nil, nil) if the file is
// unreadable due to permissions — caller treats that as "io data unavailable".
func readProcIO(pid int) (*procIO, error) {
	f, err := os.Open(procRoot + "/" + strconv.Itoa(pid) + "/io")
	if err != nil {
		// Distinguish "process gone" from "permission denied":
		// process gone → propagate (caller skips pid); permission → nil.
		if errors.Is(err, fs.ErrPermission) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	io := &procIO{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "read_bytes:"):
			v, err := parseKV(line)
			if err == nil {
				io.readBytes = v
			}
		case strings.HasPrefix(line, "write_bytes:"):
			v, err := parseKV(line)
			if err == nil {
				io.writeBytes = v
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return io, nil
}

func parseKV(line string) (uint64, error) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return 0, errors.New("malformed kv line")
	}
	return strconv.ParseUint(parts[1], 10, 64)
}

// readProcCmdline reads /proc/<pid>/cmdline and converts NUL separators
// to spaces. Returns "" for kernel threads (empty cmdline).
func readProcCmdline(pid int) (string, error) {
	data, err := os.ReadFile(procRoot + "/" + strconv.Itoa(pid) + "/cmdline")
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", nil
	}
	// Trim trailing NUL if present
	data = bytes.TrimRight(data, "\x00")
	// Replace remaining NULs with spaces
	data = bytes.ReplaceAll(data, []byte{0}, []byte{' '})
	return string(data), nil
}

// readSystemUptime parses /proc/uptime and returns the first float
// (seconds since boot) as a uint64 (seconds).
func readSystemUptime() (uint64, error) {
	data, err := os.ReadFile(procRoot + "/uptime")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, errors.New("malformed uptime")
	}
	dot := strings.IndexByte(fields[0], '.')
	if dot < 0 {
		return strconv.ParseUint(fields[0], 10, 64)
	}
	return strconv.ParseUint(fields[0][:dot], 10, 64)
}

// readCwd resolves /proc/<pid>/cwd to the target path. Returns "", nil
// if the symlink can't be read due to permission (eAccess). Other errors
// (including ENOENT from a vanished process) propagate to caller.
func readCwd(pid int) (string, error) {
	target, err := os.Readlink(procRoot + "/" + strconv.Itoa(pid) + "/cwd")
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return "", nil
		}
		return "", err
	}
	return target, nil
}

// readCgroup returns the raw contents of /proc/<pid>/cgroup. Returns "", nil
// if unreadable due to permission. The format depends on cgroup version:
//
//	v1: "12:pids:/docker/<id>" — multiple lines, one per controller
//	v2: "0::/system.slice/docker-<id>.scope" — single line, "0::" prefix
//
// Container ID extraction is handled by the caller, not this function.
func readCgroup(pid int) (string, error) {
	data, err := os.ReadFile(procRoot + "/" + strconv.Itoa(pid) + "/cgroup")
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// listPids returns all PIDs currently visible in /proc by scanning
// directory entries whose name is a positive integer.
func listPids() ([]int, error) {
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return nil, err
	}

	pids := make([]int, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}
		pids = append(pids, pid)
	}
	return pids, nil
}
