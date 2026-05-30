"""Aggregate raw collector samples into per-(host, key) summaries.

Two stages:
  1. ``compute_deltas`` — turn cumulative cpu_j / io_r / io_w into per-interval
     deltas, dropping samples where the counter went backwards (process
     restart / pid reuse).
  2. ``summarize`` — compute avg / p95 / peak / trend for each (host, key)
     across the report window.
"""
from __future__ import annotations

from dataclasses import dataclass

import numpy as np
import pandas as pd

# Linux kernel CONFIG_HZ value. Almost always 100 on the kernels we target;
# tunable here in case a deployment runs a custom kernel.
HZ = 100


def compute_deltas(df: pd.DataFrame) -> pd.DataFrame:
    """Return a copy of ``df`` with additional delta columns.

    Adds:
      - ``cpu_dj``     CPU jiffies consumed since previous sample (NaN at
                        boundaries or when counter resets)
      - ``io_dr``      delta of io_r bytes
      - ``io_dw``      delta of io_w bytes
      - ``dt_s``       seconds since previous sample for this (host, pid)
      - ``cpu_pct``    CPU% of one core consumed (HZ-aware)
                        i.e. 100% = one core fully used; >100% possible
                        if multi-threaded but kept per-process here.
    """
    if df.empty:
        return df.copy()

    # Sort so that diff() within group works in time order.
    df = df.sort_values(["host", "pid", "ts"]).reset_index(drop=True)

    grp = df.groupby(["host", "pid"], sort=False)
    df["cpu_dj"] = grp["cpu_j"].diff()
    df["io_dr"] = grp["io_r"].diff()
    df["io_dw"] = grp["io_w"].diff()
    df["dt_s"] = grp["ts"].diff()

    # Drop deltas where the counter went backwards — that means the pid was
    # reused or the process restarted, and the difference is meaningless.
    for col in ("cpu_dj", "io_dr", "io_dw"):
        df.loc[df[col] < 0, col] = np.nan

    # cpu_pct: jiffies / (seconds * HZ) * 100   per single core
    # Guard against zero-interval samples.
    safe_dt = df["dt_s"].where(df["dt_s"] > 0)
    df["cpu_pct"] = (df["cpu_dj"] / (safe_dt * HZ)) * 100.0

    return df


@dataclass
class Summary:
    """Per (host, key) summary used by the report rendering layer."""
    host: str
    key: str
    samples: int
    rss_kb_avg: float
    rss_kb_p95: float
    rss_kb_peak: float
    rss_trend_pct: float       # (last_week_avg - first_week_avg) / first * 100
    cpu_pct_avg: float
    cpu_pct_p95: float
    cpu_pct_peak: float
    io_total_bytes: float       # read+write sum across window
    io_rate_avg: float          # bytes/sec averaged across window
    io_rate_peak: float


def _safe_pct(x: pd.Series, q: float) -> float:
    if x.empty or x.dropna().empty:
        return float("nan")
    return float(np.nanpercentile(x.dropna(), q))


def _safe_max(x: pd.Series) -> float:
    if x.empty or x.dropna().empty:
        return float("nan")
    return float(np.nanmax(x))


def _safe_mean(x: pd.Series) -> float:
    if x.empty or x.dropna().empty:
        return float("nan")
    return float(np.nanmean(x))


def _trend_pct(rss: pd.Series, ts: pd.Series) -> float:
    """Trend = (mean over last third of window) vs (mean over first third).

    Using thirds rather than halves makes the signal less noisy at the
    boundary. Returns NaN if either bucket is empty.
    """
    if len(rss.dropna()) < 4:
        return float("nan")
    t_min, t_max = ts.min(), ts.max()
    if t_min == t_max:
        return float("nan")
    span = t_max - t_min
    first = rss[ts <= t_min + span / 3]
    last = rss[ts >= t_max - span / 3]
    if first.dropna().empty or last.dropna().empty:
        return float("nan")
    a = float(np.nanmean(first))
    b = float(np.nanmean(last))
    if a == 0:
        return float("nan")
    return (b - a) / a * 100.0


def aggregate_by_service(df: pd.DataFrame) -> pd.DataFrame:
    """Collapse multiple worker pids of the same cmdline_key into
    per-(host, cmdline_key, ts) totals.

    Architecture-review reports work at the *service* granularity. Readers
    want to know "celery@monitor consumes 14 GB on this host", not "each
    of the 80 celery workers uses 180 MB". Without this aggregation:

      * Charts plotting raw per-pid metrics zig-zag wildly because each
        timestamp has many y values (one per concurrent worker pid).
      * The Top10 table understates service totals — it reports per-worker
        statistics instead of per-service.

    Sum is the right aggregation for RSS / CPU% / IO bytes because those
    metrics compose across processes. (RSS double-counts shared pages —
    acceptable for capacity-planning worst case; PSS would be cleaner but
    isn't in the collected dataset.) ``dt_s`` is taken via ``max`` since
    all pids sampled at the same wall-clock ts share the same interval.
    """
    if df.empty:
        return df

    sum_cols = [c for c in ("rss_kb", "cpu_dj", "cpu_pct", "io_dr", "io_dw")
                if c in df.columns]
    agg: dict[str, str] = {c: "sum" for c in sum_cols}
    if "dt_s" in df.columns:
        agg["dt_s"] = "max"
    for c in ("comm", "cmdline", "cwd", "container"):
        if c in df.columns:
            agg[c] = "first"

    return df.groupby(
        ["host", "cmdline_key", "ts"], as_index=False, sort=False,
    ).agg(agg)


def summarize(df: pd.DataFrame) -> pd.DataFrame:
    """Group by (host, cmdline_key) and compute summary statistics."""
    if df.empty:
        return pd.DataFrame()

    summaries: list[dict] = []
    for (host, key), g in df.groupby(["host", "cmdline_key"], sort=False):
        ts = g["ts"]
        window = ts.max() - ts.min()
        # io_total: sum of per-interval deltas across the window.
        io_total = float(np.nansum(g["io_dr"].fillna(0) + g["io_dw"].fillna(0)))
        # io_rate: bytes per second averaged across the window.
        io_rate_avg = io_total / window if window > 0 else float("nan")
        # io_rate_peak: per-sample peak of (dr + dw) / dt
        with np.errstate(divide="ignore", invalid="ignore"):
            per_sample = (g["io_dr"].fillna(0) + g["io_dw"].fillna(0)) / g["dt_s"]
        io_rate_peak = _safe_max(per_sample.replace([np.inf, -np.inf], np.nan))

        summaries.append(
            dict(
                host=host,
                key=key,
                samples=len(g),
                rss_kb_avg=_safe_mean(g["rss_kb"]),
                rss_kb_p95=_safe_pct(g["rss_kb"], 95),
                rss_kb_peak=_safe_max(g["rss_kb"]),
                rss_trend_pct=_trend_pct(g["rss_kb"], ts),
                cpu_pct_avg=_safe_mean(g["cpu_pct"]),
                cpu_pct_p95=_safe_pct(g["cpu_pct"], 95),
                cpu_pct_peak=_safe_max(g["cpu_pct"]),
                io_total_bytes=io_total,
                io_rate_avg=io_rate_avg,
                io_rate_peak=io_rate_peak,
            )
        )
    return pd.DataFrame(summaries)


def top_n(summary: pd.DataFrame, host: str, metric: str, n: int = 10) -> pd.DataFrame:
    """Return the top N rows for ``host`` sorted by ``metric`` descending.

    ``metric`` must be a column name from the summary DataFrame.
    """
    if summary.empty:
        return summary
    sub = summary[summary["host"] == host]
    return sub.sort_values(metric, ascending=False).head(n).reset_index(drop=True)
