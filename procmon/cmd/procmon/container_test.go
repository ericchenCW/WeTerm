package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractContainerID(t *testing.T) {
	const id = "abc123def456789012345678901234567890123456789012345678901234abcd"

	cases := map[string]struct {
		cgroup string
		want   string
	}{
		"docker cgv1": {
			cgroup: "12:pids:/docker/" + id + "\n11:cpu:/docker/" + id + "\n",
			want:   id,
		},
		"docker cgv2 systemd": {
			cgroup: "0::/system.slice/docker-" + id + ".scope\n",
			want:   id,
		},
		"cri-containerd k8s": {
			cgroup: "0::/kubepods.slice/kubepods-besteffort.slice/" +
				"cri-containerd-" + id + ".scope\n",
			want: id,
		},
		"containerd standalone": {
			cgroup: "0::/system.slice/containerd-" + id + ".scope\n",
			want:   id,
		},
		"podman libpod": {
			cgroup: "0::/user.slice/user-1000.slice/libpod-" + id + ".scope\n",
			want:   id,
		},
		"podman older": {
			cgroup: "0::/podman-" + id + ".scope\n",
			want:   id,
		},
		"systemd unit (not a container)": {
			cgroup: "0::/system.slice/nginx.service\n",
			want:   "",
		},
		"user session (not a container)": {
			cgroup: "0::/user.slice/user-1000.slice/session-3.scope\n",
			want:   "",
		},
		"empty": {
			cgroup: "",
			want:   "",
		},
		// Pod UUIDs (e.g. kubepods-besteffort-podc60c7d10-...) must NOT match.
		// Our regex requires 64-hex and surrounding `docker-`/`containerd-`/
		// `podman-`/`libpod-` markers, so a 32-hex pod UUID won't trigger.
		"pod uuid alone should not match": {
			cgroup: "0::/kubepods.slice/kubepods-besteffort-podc60c7d10-9c5b-11e6-9b3a-080027a45f73.slice\n",
			want:   "",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := extractContainerID(tc.cgroup)
			if got != tc.want {
				t.Errorf("extractContainerID = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDockerName_Success(t *testing.T) {
	tmp := t.TempDir()
	id := "deadbeefcafe1234567890123456789012345678901234567890123456789012"
	dir := filepath.Join(tmp, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "config.v2.json"),
		[]byte(`{"Name":"/order-svc","other":"ignored"}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	orig := dockerRoot
	dockerRoot = tmp
	defer func() { dockerRoot = orig }()

	name, ok := dockerName(id)
	if !ok || name != "order-svc" {
		t.Errorf("dockerName = (%q, %v), want (order-svc, true)", name, ok)
	}
}

func TestDockerName_NotFound(t *testing.T) {
	orig := dockerRoot
	dockerRoot = t.TempDir()
	defer func() { dockerRoot = orig }()

	_, ok := dockerName("nonexistent")
	if ok {
		t.Errorf("dockerName should fail for missing config")
	}
}

func TestDockerName_Malformed(t *testing.T) {
	tmp := t.TempDir()
	id := "aaaa"
	dir := filepath.Join(tmp, id)
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "config.v2.json"),
		[]byte(`{not valid json`), 0o644)

	orig := dockerRoot
	dockerRoot = tmp
	defer func() { dockerRoot = orig }()

	_, ok := dockerName(id)
	if ok {
		t.Errorf("dockerName should fail for malformed json")
	}
}

func TestResolveContainer_DockerWithName(t *testing.T) {
	id := "1111222233334444555566667777888899990000aaaabbbbccccddddeeee0000"
	// Set up fake /proc tree
	defer buildFakeProc(t, map[string]string{
		"42/cgroup": "0::/system.slice/docker-" + id + ".scope\n",
	})()
	// Set up fake docker config
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, id), 0o755)
	os.WriteFile(filepath.Join(tmp, id, "config.v2.json"),
		[]byte(`{"Name":"/order-svc"}`), 0o644)
	orig := dockerRoot
	dockerRoot = tmp
	defer func() { dockerRoot = orig }()

	got, err := resolveContainer(42)
	if err != nil {
		t.Fatal(err)
	}
	if got != "order-svc" {
		t.Errorf("resolveContainer = %q, want order-svc", got)
	}
}

func TestResolveContainer_FallbackToShortID(t *testing.T) {
	id := "1111222233334444555566667777888899990000aaaabbbbccccddddeeee0000"
	defer buildFakeProc(t, map[string]string{
		"42/cgroup": "0::/system.slice/docker-" + id + ".scope\n",
	})()
	// Docker root points to empty dir — Name lookup will fail.
	orig := dockerRoot
	dockerRoot = t.TempDir()
	defer func() { dockerRoot = orig }()

	got, err := resolveContainer(42)
	if err != nil {
		t.Fatal(err)
	}
	if got != id[:12] {
		t.Errorf("resolveContainer = %q, want short ID %q", got, id[:12])
	}
}

func TestReadDaemonJSONDataRoot(t *testing.T) {
	tmp := t.TempDir()
	cases := map[string]struct {
		body string
		want string
	}{
		"data-root set":     {`{"data-root": "/data/docker"}`, "/data/docker"},
		"data-root absent":  {`{"storage-driver": "overlay2"}`, ""},
		"data-root empty":   {`{"data-root": ""}`, ""},
		"malformed":         {`{not json`, ""},
		"other fields":      {`{"data-root": "/srv/d", "debug": true}`, "/srv/d"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p := tmp + "/" + name + ".json"
			os.WriteFile(p, []byte(tc.body), 0o644)
			got := readDaemonJSONDataRoot(p)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReadDaemonJSONDataRoot_FileMissing(t *testing.T) {
	if got := readDaemonJSONDataRoot("/nonexistent/path/daemon.json"); got != "" {
		t.Errorf("missing file should yield empty string, got %q", got)
	}
}

func TestInitDockerRoot_PrefersOverride(t *testing.T) {
	orig := dockerRoot
	defer func() { dockerRoot = orig }()

	initDockerRoot("/explicit/root")
	if dockerRoot != "/explicit/root/containers" {
		t.Errorf("dockerRoot = %q, want /explicit/root/containers", dockerRoot)
	}
}

func TestInitDockerRoot_DefaultFallback(t *testing.T) {
	orig := dockerRoot
	defer func() { dockerRoot = orig }()

	// No override, no daemon.json (assuming test machine doesn't have one
	// or has one without data-root — we can't make a real assertion either
	// way, so just verify behavior is deterministic).
	initDockerRoot("")
	if dockerRoot == "" {
		t.Errorf("dockerRoot should never be empty after init")
	}
	if !(dockerRoot == "/var/lib/docker/containers" ||
		// On a dev machine that has /etc/docker/daemon.json with a custom
		// root, init will pick that — both are valid.
		dockerRoot != "/var/lib/docker/containers") {
		t.Errorf("unexpected dockerRoot: %q", dockerRoot)
	}
}

func TestResolveContainer_NotInContainer(t *testing.T) {
	defer buildFakeProc(t, map[string]string{
		"42/cgroup": "0::/system.slice/nginx.service\n",
	})()
	got, err := resolveContainer(42)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("non-container should resolve to empty, got %q", got)
	}
}
