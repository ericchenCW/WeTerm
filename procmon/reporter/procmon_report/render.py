"""Render the aggregated summary into a self-contained HTML report."""
from __future__ import annotations

import datetime as dt
import math
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import jinja2
import numpy as np
import pandas as pd

from .aggregate import top_n
from .normalize import redact_cmdline
from .plot import multi_line_chart_html, plotly_runtime_script

# Threshold for highlighting a host's peak memory utilization in red.
_MEM_WARN_PCT = 85.0


# ---------------------------------------------------------------------------
# Formatting helpers — keep numbers compact and human-readable.

def _fmt_bytes(b: float) -> str:
    if b is None or (isinstance(b, float) and math.isnan(b)):
        return "—"
    units = ["B", "KB", "MB", "GB", "TB"]
    n = float(b)
    for u in units:
        if abs(n) < 1024:
            return f"{n:.1f}{u}" if n < 100 else f"{n:.0f}{u}"
        n /= 1024
    return f"{n:.1f}PB"


def _fmt_kb_as_mem(kb: float) -> str:
    if kb is None or (isinstance(kb, float) and math.isnan(kb)):
        return "—"
    return _fmt_bytes(kb * 1024)


def _fmt_pct(p: float) -> str:
    if p is None or (isinstance(p, float) and math.isnan(p)):
        return "—"
    return f"{p:.1f}%"


def _fmt_rate(bps: float) -> str:
    if bps is None or (isinstance(bps, float) and math.isnan(bps)):
        return "—"
    return f"{_fmt_bytes(bps)}/s"


def _fmt_trend(pct: float) -> tuple[str, str]:
    """Return (text, css_class) for a trend percentage."""
    if pct is None or (isinstance(pct, float) and math.isnan(pct)):
        return ("→ 平稳", "")
    if pct >= 10:
        return (f"↗ +{pct:.0f}%", "up")
    if pct <= -10:
        return (f"↘ {pct:.0f}%", "down")
    return ("→ 平稳", "")


# ---------------------------------------------------------------------------
# Host-level metadata derivation
#
# We can't ssh out to each host at report time, so we infer specs from the
# samples we have. This is a best-effort number — when /proc/meminfo isn't
# in the sample (it isn't; collector only records per-process data), we
# approximate total memory by the sum of all process RSS at the moment of
# the highest observed total. Imperfect but enough for "is this host
# under pressure" signal in the overview chips.

@dataclass
class HostMeta:
    name: str
    cores: int | None
    mem_total_gb: float | None
    peak_mem_pct: float | None
    ip: str | None = None
    roles: str | None = None
    disk_gb: float | None = None
    mem_is_real: bool = False  # True when mem_total_gb came from host-meta, not inferred


def _derive_host_meta(host_df: pd.DataFrame, spec: dict | None = None) -> HostMeta:
    name = host_df["host"].iloc[0]

    # Peak total = largest per-timestamp RSS sum we observed. NOTE: summing
    # per-process RSS double-counts shared pages, so it overstates true
    # physical usage — fine as a relative signal, not an absolute one.
    per_ts_rss = host_df.groupby("ts")["rss_kb"].sum()
    peak_total_kb = float(per_ts_rss.max()) if not per_ts_rss.empty else 0.0

    # Real specs from host-meta.json take precedence: use the true total
    # memory as the denominator for the utilization figure, and surface
    # ip / cores / disk / roles.
    if spec:
        mem_total_gb = spec.get("mem_gb")
        peak_pct = (peak_total_kb / 1024 / 1024 / mem_total_gb * 100
                    if mem_total_gb else None)
        return HostMeta(
            name=name, cores=spec.get("cores"), mem_total_gb=mem_total_gb,
            peak_mem_pct=peak_pct, ip=spec.get("ip"), roles=spec.get("roles"),
            disk_gb=spec.get("disk_gb"), mem_is_real=mem_total_gb is not None,
        )

    # Fallback: no real spec — infer total memory by rounding the observed
    # peak up to the nearest 4 GB step. Marked as an estimate (mem_is_real=False).
    if per_ts_rss.empty or peak_total_kb <= 0:
        return HostMeta(name=name, cores=None, mem_total_gb=None, peak_mem_pct=None)
    gb = peak_total_kb / 1024 / 1024
    mem_total_gb = max(4.0, 4.0 * math.ceil(gb / 4.0))
    peak_pct = peak_total_kb / 1024 / 1024 / mem_total_gb * 100
    return HostMeta(name=name, cores=None, mem_total_gb=mem_total_gb,
                    peak_mem_pct=peak_pct)


# ---------------------------------------------------------------------------
# Render

def _row_for_mem(row: pd.Series) -> dict[str, Any]:
    text, cls = _fmt_trend(row["rss_trend_pct"])
    return dict(
        key=row["key"],
        avg=_fmt_kb_as_mem(row["rss_kb_avg"]),
        p95=_fmt_kb_as_mem(row["rss_kb_p95"]),
        peak=_fmt_kb_as_mem(row["rss_kb_peak"]),
        trend=text,
        trend_class=cls,
    )


def _row_for_cpu(row: pd.Series) -> dict[str, Any]:
    return dict(
        key=row["key"],
        avg=_fmt_pct(row["cpu_pct_avg"]),
        p95=_fmt_pct(row["cpu_pct_p95"]),
        peak=_fmt_pct(row["cpu_pct_peak"]),
    )


def _row_for_io(row: pd.Series) -> dict[str, Any]:
    return dict(
        key=row["key"],
        total=_fmt_bytes(row["io_total_bytes"]),
        avg_rate=_fmt_rate(row["io_rate_avg"]),
        peak_rate=_fmt_rate(row["io_rate_peak"]),
    )


def _build_host_charts(host_df: pd.DataFrame, mem_top: pd.DataFrame,
                       cpu_top: pd.DataFrame, io_top: pd.DataFrame,
                       *, chart_top_n: int = 8) -> dict[str, str]:
    """Return three multi-line plotly charts for a host: memory / CPU / IO.

    Each chart picks the top-N services by that metric (so the legend on
    each chart matches what's directly above in its Top10 table). Cap at
    ``chart_top_n`` (default 8) so the legend stays readable.
    """
    mem_keys = list(mem_top["key"])[:chart_top_n]
    cpu_keys = list(cpu_top["key"])[:chart_top_n]
    io_keys = list(io_top["key"])[:chart_top_n]

    # Need IO rate per-sample for the chart. compute_deltas left io_dr/io_dw
    # and dt_s on the frame; we synthesize a transient column here.
    host_df = host_df.copy()
    with np.errstate(divide="ignore", invalid="ignore"):
        host_df["_io_rate"] = (
            host_df["io_dr"].fillna(0) + host_df["io_dw"].fillna(0)
        ) / host_df["dt_s"]
    host_df["_io_rate"] = host_df["_io_rate"].replace([np.inf, -np.inf], np.nan)

    return dict(
        memory=multi_line_chart_html(
            host_df, mem_keys, value_col="rss_kb",
            value_transform=lambda kb: kb / 1024.0,  # KB → MB
            y_title="RSS (MB)", hover_unit="MB",
        ),
        cpu=multi_line_chart_html(
            host_df, cpu_keys, value_col="cpu_pct",
            y_title="CPU (%)", hover_unit="%",
        ),
        io=multi_line_chart_html(
            host_df, io_keys, value_col="_io_rate",
            value_transform=lambda b: (b or 0) / 1024.0 / 1024.0,  # B/s → MB/s
            y_title="IO (MB/s)", hover_unit="MB/s",
            hover_fmt=".2f",
        ),
    )


def _key_map(df: pd.DataFrame) -> list[dict]:
    """Build the cmdline_key audit table — unique rows over (comm, cmdline, cwd, container)."""
    seen: dict[tuple[str, str, str, str], str] = {}
    for _, row in df.iterrows():
        comm = str(row.get("comm", ""))
        cmd = str(row.get("cmdline", ""))
        cwd = str(row.get("cwd") or "") if row.get("cwd") is not None else ""
        container = str(row.get("container") or "") if row.get("container") is not None else ""
        key = str(row.get("cmdline_key", ""))
        tup = (comm, cmd, cwd, container)
        if tup not in seen:
            seen[tup] = key
    out = []
    for (comm, cmd, cwd, container), key in sorted(seen.items()):
        out.append(dict(
            comm=comm,
            cmdline=redact_cmdline(cmd),
            cwd=cwd or "—",
            container=container or "—",
            key=key,
        ))
    return out


def _build_roles_overview(host_meta: dict) -> dict:
    """Turn the host-meta spec map into the role-distribution overview table:
    one row per host (in meta order) plus a totals row."""
    rows = []
    tot_cores = tot_mem = tot_disk = 0
    for spec in host_meta.values():
        rows.append(dict(
            ip=spec.get("ip") or "—",
            roles=spec.get("roles") or "—",
            cores=spec.get("cores"),
            mem_gb=spec.get("mem_gb"),
            disk_gb=spec.get("disk_gb"),
        ))
        tot_cores += spec.get("cores") or 0
        tot_mem += spec.get("mem_gb") or 0
        tot_disk += spec.get("disk_gb") or 0
    return dict(rows=rows, total=dict(cores=tot_cores, mem_gb=tot_mem, disk_gb=tot_disk))


def render_html(
    df: pd.DataFrame,
    summary: pd.DataFrame,
    *,
    df_raw: pd.DataFrame | None = None,
    sample_interval_s: int = 60,
    top_n_value: int = 10,
    template_dir: Path | None = None,
    host_meta: dict | None = None,
    overview_note: str | None = None,
    plan: dict | None = None,
) -> str:
    """Build the final HTML string.

    ``df``         per-service frame (aggregated across pids by ts); drives
                   host loop, charts, and host metadata.
    ``df_raw``     optional pre-aggregation frame; if provided, the
                   cmdline_key audit appendix uses it to surface every
                   distinct (comm, cmdline, cwd, container) variant that
                   contributed to a key. Falls back to ``df`` when omitted.
    ``host_meta``  optional ``{host: {ip, cores, mem_gb, disk_gb, roles}}``
                   map of real specs. When given it replaces the inferred
                   per-host specs, fixes the memory utilization denominator,
                   sets the host display order, and renders an overview table.
    ``overview_note`` optional caption shown under the overview table.
    """

    template_dir = template_dir or (Path(__file__).resolve().parent.parent / "templates")
    env = jinja2.Environment(
        loader=jinja2.FileSystemLoader(template_dir),
        autoescape=jinja2.select_autoescape(["html"]),
        undefined=jinja2.StrictUndefined,
    )
    tpl = env.get_template("report.html.j2")

    ts_min = df["ts"].min() if not df.empty else 0
    ts_max = df["ts"].max() if not df.empty else 0
    window = dict(
        start=dt.datetime.fromtimestamp(int(ts_min)).strftime("%Y-%m-%d"),
        end=dt.datetime.fromtimestamp(int(ts_max)).strftime("%Y-%m-%d"),
        sample_interval_s=sample_interval_s,
        total_samples=int(len(df)),
        generated_at=dt.datetime.now().strftime("%Y-%m-%d %H:%M"),
    )

    # Host display order: follow host-meta order (matching the overview
    # table's row order) when provided, then append any leftover hosts
    # alphabetically. Without meta, fall back to alphabetical.
    present = list(df["host"].unique())
    if host_meta:
        ordered = [h for h in host_meta if h in present]
        ordered += sorted(h for h in present if h not in host_meta)
    else:
        ordered = sorted(present)

    hosts_view = []
    for host_name in ordered:
        host_df = df[df["host"] == host_name]
        meta = _derive_host_meta(host_df, (host_meta or {}).get(host_name))

        mem_top = top_n(summary, host_name, "rss_kb_peak", top_n_value)
        cpu_top = top_n(summary, host_name, "cpu_pct_avg", top_n_value)
        io_top = top_n(summary, host_name, "io_total_bytes", top_n_value)

        charts = _build_host_charts(host_df, mem_top, cpu_top, io_top)
        hosts_view.append(dict(
            name=meta.name,
            ip=meta.ip,
            cores=meta.cores,
            mem_total_gb=meta.mem_total_gb,
            mem_is_real=meta.mem_is_real,
            peak_mem_pct=meta.peak_mem_pct,
            mem_top=[_row_for_mem(r) for _, r in mem_top.iterrows()],
            cpu_top=[_row_for_cpu(r) for _, r in cpu_top.iterrows()],
            io_top=[_row_for_io(r) for _, r in io_top.iterrows()],
            charts=charts,
        ))

    return tpl.render(
        window=window,
        hosts=hosts_view,
        roles_overview=_build_roles_overview(host_meta) if host_meta else None,
        overview_note=overview_note,
        plan=plan,
        key_map=_key_map(df_raw if df_raw is not None else df),
        plotly_runtime=plotly_runtime_script(),
    )
