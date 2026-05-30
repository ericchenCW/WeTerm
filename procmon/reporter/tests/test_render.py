"""Smoke test the HTML render path — no semantic assertions beyond presence."""
import pandas as pd
import pytest

from procmon_report.aggregate import compute_deltas, summarize
from procmon_report.normalize import normalize_key
from procmon_report.render import render_html


def _make_synthetic(n_hosts=2, services_per_host=3, days=2, sample_s=60):
    """Build a small but realistic raw DataFrame."""
    rows = []
    for h in range(n_hosts):
        host = f"host-{chr(ord('a') + h)}"
        for svc in range(services_per_host):
            comm = "java" if svc == 0 else "mysqld" if svc == 1 else "nginx"
            cmd = (f"java -jar svc-{h}-{svc}.jar" if comm == "java"
                   else f"{comm} --port=33{svc:02d}")
            pid = 1000 + h * 100 + svc
            cpu = 0
            io_r = 0
            io_w = 0
            for t in range(0, days * 24 * 3600, sample_s):
                cpu += 30 + svc * 10
                io_r += 1024 * (svc + 1)
                io_w += 512 * (svc + 1)
                rows.append({
                    "ts": t, "host": host, "pid": pid, "comm": comm,
                    "cmdline": cmd, "cpu_j": cpu,
                    "rss_kb": 100_000 + svc * 50_000, "io_r": io_r,
                    "io_w": io_w, "uptime_s": t + 100,
                })
    df = pd.DataFrame(rows)
    df["cmdline_key"] = df.apply(
        lambda r: normalize_key(str(r["comm"]), str(r["cmdline"])), axis=1
    )
    return df


def test_render_smoke():
    raw = _make_synthetic()
    df = compute_deltas(raw)
    s = summarize(df)
    html = render_html(df, s)
    assert "<html" in html.lower()
    assert "host-a" in html
    assert "host-b" in html
    # All three host services should be present in at least one top-10
    assert "java:svc-0-0" in html
    assert "mysqld" in html
    assert "nginx" in html
    # Plotly runtime must be embedded (we don't depend on a CDN).
    # The minified library defines a global named `Plotly`; checking for
    # its presence in the JS bundle is a cheap sanity check.
    assert "Plotly" in html
    # Each host should have three chart divs (memory, cpu, io).
    assert html.count('class="chart-block"') == 2 * 3  # 2 hosts × 3 charts
    # Appendix should default to collapsed and list the key map
    assert '<details class="appendix"' in html
    assert "归一化映射" in html


def test_render_empty_df_returns_empty_friendly():
    # No data — render should not crash, just produce a near-empty page.
    df = pd.DataFrame(columns=["ts", "host", "pid", "comm", "cmdline",
                                "cmdline_key", "cpu_j", "rss_kb", "io_r",
                                "io_w", "uptime_s", "cpu_dj", "io_dr",
                                "io_dw", "dt_s", "cpu_pct"])
    s = pd.DataFrame()
    # With empty df, the renderer is allowed to produce a minimal page.
    # We don't enforce a specific message, just that it doesn't blow up.
    html = render_html(df, s)
    assert "<html" in html.lower()


@pytest.mark.parametrize("metric", ["rss_kb_peak", "cpu_pct_avg", "io_total_bytes"])
def test_render_metrics_all_have_values(metric):
    raw = _make_synthetic()
    df = compute_deltas(raw)
    s = summarize(df)
    # Sanity: the summary has the metric and no all-NaN host
    for host in s["host"].unique():
        sub = s[s["host"] == host]
        # At least one row should have a real number
        assert sub[metric].dropna().shape[0] > 0
