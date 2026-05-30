"""Command-line entry point: ``python -m procmon_report ...``."""
from __future__ import annotations

import argparse
import sys
from pathlib import Path

from . import __version__
from .aggregate import compute_deltas, summarize
from .loader import load_dir
from .normalize import normalize_key, redact_cmdline


def _build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(
        prog="procmon-report",
        description="Generate an HTML resource-usage report from procmon JSONL data.",
    )
    p.add_argument("--data-dir", required=True, type=Path,
                   help="directory containing {host}-YYYY-MM-DD.jsonl files")
    p.add_argument("--out", default=Path("report.html"), type=Path,
                   help="output HTML file (default: report.html)")
    p.add_argument("--top-n", type=int, default=10,
                   help="how many entries per Top table (default: 10)")
    p.add_argument("--sample-interval", type=int, default=60,
                   help="nominal sample interval in seconds for the header (default: 60)")
    p.add_argument("--host-meta", type=Path, default=None,
                   help="optional JSON file with real host specs (ip / cores / "
                        "mem_gb / disk_gb / roles) to replace inferred values "
                        "and add a role-distribution overview table")
    p.add_argument("--plan", type=Path, default=None,
                   help="optional JSON file with a target-architecture assessment "
                        "(per-machine specs + verdict) to render an evaluation section")
    p.add_argument("--dump-keys", action="store_true",
                   help="print the (comm, cmdline, key) mapping and exit "
                        "without rendering — useful for tuning normalization rules")
    p.add_argument("--version", action="version", version=f"procmon-report {__version__}")
    return p


def main(argv: list[str] | None = None) -> int:
    args = _build_parser().parse_args(argv)

    if not args.data_dir.is_dir():
        print(f"error: not a directory: {args.data_dir}", file=sys.stderr)
        return 2

    df, stats = load_dir(args.data_dir)
    print(
        f"loaded files={stats.files} lines={stats.lines} bad_lines={stats.bad_lines}",
        file=sys.stderr,
    )

    if df.empty:
        print("error: no usable records found in data-dir", file=sys.stderr)
        return 1

    # Derive cmdline_key once for all records. The lambda passes cwd /
    # container as keyword args; both are guaranteed to be present as
    # columns (loader backfills None for older records).
    def _key(r):
        return normalize_key(
            str(r["comm"]),
            str(r["cmdline"]),
            cwd=r.get("cwd"),
            container=r.get("container"),
        )

    df["cmdline_key"] = df.apply(_key, axis=1)

    if args.dump_keys:
        seen: set[tuple[str, str, str, str]] = set()
        for _, r in df.iterrows():
            tup = (
                str(r["comm"]),
                str(r["cmdline"]),
                str(r.get("cwd") or ""),
                str(r.get("container") or ""),
            )
            if tup in seen:
                continue
            seen.add(tup)
            comm, cmd, cwd, container = tup
            key = normalize_key(comm, cmd, cwd=cwd or None, container=container or None)
            print(f"{comm}\t{redact_cmdline(cmd)}\t{cwd or '-'}\t{container or '-'}\t{key}")
        return 0

    df = compute_deltas(df)

    # Optional real host specs (ip / cores / mem / disk / roles). When absent
    # the renderer falls back to inferring specs from the samples.
    host_meta = None
    overview_note = None
    if args.host_meta is not None:
        import json
        doc = json.loads(args.host_meta.read_text(encoding="utf-8"))
        host_meta = doc.get("hosts", doc)  # allow a flat {host: spec} doc too
        overview_note = doc.get("note")

    plan = None
    if args.plan is not None:
        import json
        plan = json.loads(args.plan.read_text(encoding="utf-8"))

    # Aggregate per-pid rows into per-service totals before computing stats.
    # See aggregate_by_service docstring for rationale.
    from .aggregate import aggregate_by_service
    df_agg = aggregate_by_service(df)
    summary = summarize(df_agg)

    # Lazy import to keep ``--dump-keys`` startup cheap.
    from .render import render_html
    html = render_html(
        df_agg, summary,
        df_raw=df,  # raw frame retains all pid/cmdline variants for the audit appendix
        sample_interval_s=args.sample_interval,
        top_n_value=args.top_n,
        host_meta=host_meta,
        overview_note=overview_note,
        plan=plan,
    )
    args.out.write_text(html, encoding="utf-8")
    print(f"wrote {args.out} ({len(html):,} bytes)", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
