"""Load and parse procmon JSONL files from a directory.

Filename convention: ``{hostname}-YYYY-MM-DD.jsonl``. The hostname is
recovered from the filename rather than the record body so that we don't
rely on collectors having identical hostnames across samples.
"""
from __future__ import annotations

import json
import re
import sys
from dataclasses import dataclass
from pathlib import Path

import pandas as pd

# host can contain hyphens, so we anchor on the trailing YYYY-MM-DD.jsonl.
_FILENAME_RE = re.compile(r"^(?P<host>.+)-(?P<date>\d{4}-\d{2}-\d{2})\.jsonl$")


@dataclass
class LoadStats:
    files: int = 0
    lines: int = 0
    bad_lines: int = 0


def load_dir(data_dir: Path) -> tuple[pd.DataFrame, LoadStats]:
    """Load every ``*.jsonl`` file matching the host-date convention.

    Returns a DataFrame with the raw record fields plus a derived ``host``
    column, and a LoadStats summary written to stderr by the caller.
    """
    stats = LoadStats()
    rows: list[dict] = []

    files = sorted(p for p in data_dir.iterdir() if p.is_file() and p.suffix == ".jsonl")
    for path in files:
        m = _FILENAME_RE.match(path.name)
        if not m:
            print(f"loader: skip non-matching filename: {path.name}", file=sys.stderr)
            continue
        host = m.group("host")
        stats.files += 1
        with path.open("r", encoding="utf-8") as f:
            for line_no, line in enumerate(f, 1):
                line = line.strip()
                if not line:
                    continue
                stats.lines += 1
                try:
                    rec = json.loads(line)
                except json.JSONDecodeError:
                    stats.bad_lines += 1
                    continue
                # Trust the filename's host (it's the source of truth for
                # which file the record came from) and overwrite the
                # in-record host field for consistency.
                rec["host"] = host
                rows.append(rec)

    if not rows:
        return pd.DataFrame(), stats

    df = pd.DataFrame.from_records(rows)
    # Normalize types — pandas would otherwise leave io_r/io_w as object
    # because of the nulls.
    df["ts"] = pd.to_numeric(df["ts"], downcast="integer")
    df["pid"] = pd.to_numeric(df["pid"], downcast="integer")
    df["cpu_j"] = pd.to_numeric(df["cpu_j"], errors="coerce")
    df["rss_kb"] = pd.to_numeric(df["rss_kb"], errors="coerce")
    df["io_r"] = pd.to_numeric(df["io_r"], errors="coerce")
    df["io_w"] = pd.to_numeric(df["io_w"], errors="coerce")
    df["uptime_s"] = pd.to_numeric(df["uptime_s"], errors="coerce")
    df["dt"] = pd.to_datetime(df["ts"], unit="s")
    # Optional fields — backfill as None for records produced by older
    # collectors that didn't emit them. When the column exists but some
    # rows are missing the key (mixed old + new records in the same file),
    # pandas writes NaN — normalize that to None too so downstream truthy
    # checks like `if cwd:` work consistently.
    for opt in ("cwd", "container"):
        if opt not in df.columns:
            df[opt] = None
        else:
            df[opt] = df[opt].astype(object).where(df[opt].notna(), None)
    return df, stats
