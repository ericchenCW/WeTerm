import json
from pathlib import Path

from procmon_report.loader import load_dir


def _write_jsonl(path: Path, records):
    with path.open("w") as f:
        for r in records:
            f.write(json.dumps(r) + "\n")


def test_load_basic(tmp_path: Path):
    _write_jsonl(
        tmp_path / "host-a-2026-05-21.jsonl",
        [
            {"ts": 100, "host": "host-a", "pid": 1, "comm": "init",
             "cmdline": "init", "cpu_j": 50, "rss_kb": 1024,
             "io_r": 0, "io_w": 0, "uptime_s": 100},
            {"ts": 160, "host": "host-a", "pid": 1, "comm": "init",
             "cmdline": "init", "cpu_j": 60, "rss_kb": 1024,
             "io_r": 100, "io_w": 200, "uptime_s": 160},
        ],
    )
    df, stats = load_dir(tmp_path)
    assert stats.files == 1
    assert stats.lines == 2
    assert stats.bad_lines == 0
    assert len(df) == 2
    assert set(df["host"].unique()) == {"host-a"}


def test_load_skips_corrupt_lines(tmp_path: Path):
    f = tmp_path / "host-x-2026-05-21.jsonl"
    with f.open("w") as fh:
        fh.write(json.dumps({"ts": 1, "host": "host-x", "pid": 1, "comm": "x",
                             "cmdline": "x", "cpu_j": 1, "rss_kb": 1,
                             "io_r": 0, "io_w": 0, "uptime_s": 1}) + "\n")
        fh.write("this is not json\n")
        fh.write(json.dumps({"ts": 2, "host": "host-x", "pid": 1, "comm": "x",
                             "cmdline": "x", "cpu_j": 2, "rss_kb": 1,
                             "io_r": 0, "io_w": 0, "uptime_s": 2}) + "\n")
    df, stats = load_dir(tmp_path)
    assert stats.bad_lines == 1
    assert len(df) == 2


def test_load_skips_non_matching_filenames(tmp_path: Path):
    (tmp_path / "README.txt").write_text("ignore me")
    (tmp_path / "garbage.jsonl").write_text("")
    (tmp_path / "host-good-2026-05-21.jsonl").write_text(
        json.dumps({"ts": 1, "host": "host-good", "pid": 1, "comm": "x",
                    "cmdline": "x", "cpu_j": 1, "rss_kb": 1,
                    "io_r": 0, "io_w": 0, "uptime_s": 1}) + "\n"
    )
    df, stats = load_dir(tmp_path)
    assert stats.files == 1  # only the matching one
    assert len(df) == 1


def test_load_overrides_host_from_filename(tmp_path: Path):
    # If the in-record host disagrees with the filename, the filename wins.
    (tmp_path / "host-a-2026-05-21.jsonl").write_text(
        json.dumps({"ts": 1, "host": "WRONG", "pid": 1, "comm": "x",
                    "cmdline": "x", "cpu_j": 1, "rss_kb": 1,
                    "io_r": 0, "io_w": 0, "uptime_s": 1}) + "\n"
    )
    df, _ = load_dir(tmp_path)
    assert df["host"].iloc[0] == "host-a"


def test_load_empty_dir(tmp_path: Path):
    df, stats = load_dir(tmp_path)
    assert df.empty
    assert stats.files == 0


def test_load_backfills_optional_columns_for_old_records(tmp_path: Path):
    # Records from a pre-v2 collector have no cwd / container fields.
    # The loader must still produce a DataFrame with those columns set
    # to None so downstream code can rely on them existing.
    f = tmp_path / "host-old-2026-05-21.jsonl"
    f.write_text(
        json.dumps({
            "ts": 1, "host": "host-old", "pid": 1, "comm": "x",
            "cmdline": "x", "cpu_j": 1, "rss_kb": 1,
            "io_r": 0, "io_w": 0, "uptime_s": 1,
        }) + "\n"
    )
    df, _ = load_dir(tmp_path)
    assert "cwd" in df.columns
    assert "container" in df.columns
    assert df["cwd"].iloc[0] is None
    assert df["container"].iloc[0] is None


def test_load_mixes_old_and_new_records(tmp_path: Path):
    # New collector emits cwd/container; old one doesn't. Both should load.
    f = tmp_path / "host-mixed-2026-05-21.jsonl"
    with f.open("w") as fh:
        # old-shape line
        fh.write(json.dumps({
            "ts": 1, "host": "host-mixed", "pid": 1, "comm": "x",
            "cmdline": "x", "cpu_j": 1, "rss_kb": 1,
            "io_r": 0, "io_w": 0, "uptime_s": 1,
        }) + "\n")
        # new-shape line with explicit container
        fh.write(json.dumps({
            "ts": 2, "host": "host-mixed", "pid": 2, "comm": "y",
            "cmdline": "y", "cwd": "/srv/y", "container": "ctr-y",
            "cpu_j": 1, "rss_kb": 1, "io_r": 0, "io_w": 0, "uptime_s": 2,
        }) + "\n")
    df, _ = load_dir(tmp_path)
    # The pid=1 row backfills as None; the pid=2 row keeps real values.
    by_pid = {r["pid"]: r for _, r in df.iterrows()}
    assert by_pid[1]["cwd"] is None
    assert by_pid[1]["container"] is None
    assert by_pid[2]["cwd"] == "/srv/y"
    assert by_pid[2]["container"] == "ctr-y"
