import numpy as np
import pandas as pd

from procmon_report.aggregate import (
    aggregate_by_service, compute_deltas, summarize, top_n, HZ,
)


def _df(rows):
    return pd.DataFrame(rows)


def test_compute_deltas_basic():
    # 60s interval, 300 jiffies in 60s on HZ=100  =>  5% of one core
    df = _df([
        {"host": "h", "pid": 1, "ts": 0,  "comm": "x", "cmdline": "x",
         "cmdline_key": "x", "cpu_j": 1000, "rss_kb": 100, "io_r": 0, "io_w": 0},
        {"host": "h", "pid": 1, "ts": 60, "comm": "x", "cmdline": "x",
         "cmdline_key": "x", "cpu_j": 1300, "rss_kb": 100, "io_r": 50, "io_w": 50},
    ])
    out = compute_deltas(df)
    second_row = out.iloc[1]
    assert second_row["cpu_dj"] == 300
    assert second_row["dt_s"] == 60
    # 300 / (60 * 100) * 100 = 5%
    assert abs(second_row["cpu_pct"] - 5.0) < 1e-6
    assert second_row["io_dr"] == 50
    assert second_row["io_dw"] == 50


def test_compute_deltas_counter_reset_dropped():
    # Counter decreasing = process restart / pid reuse. Delta must be dropped.
    df = _df([
        {"host": "h", "pid": 1, "ts": 0,  "comm": "x", "cmdline": "x",
         "cmdline_key": "x", "cpu_j": 5000, "rss_kb": 100, "io_r": 1000, "io_w": 1000},
        {"host": "h", "pid": 1, "ts": 60, "comm": "x", "cmdline": "x",
         "cmdline_key": "x", "cpu_j": 100, "rss_kb": 100, "io_r": 50, "io_w": 50},
    ])
    out = compute_deltas(df)
    assert np.isnan(out.iloc[1]["cpu_dj"])
    assert np.isnan(out.iloc[1]["cpu_pct"])
    assert np.isnan(out.iloc[1]["io_dr"])
    assert np.isnan(out.iloc[1]["io_dw"])


def test_compute_deltas_empty():
    out = compute_deltas(pd.DataFrame())
    assert out.empty


def test_summarize_peak_and_p95():
    # Synthetic series with one outlier — peak should pick the outlier
    # but p95 should ignore it (roughly).
    ts = list(range(0, 6000, 60))  # 100 samples
    rss = [1000] * 99 + [9999]      # one outlier
    rows = []
    for t, r in zip(ts, rss):
        rows.append({
            "host": "h", "pid": 1, "ts": t, "comm": "java",
            "cmdline": "java -jar foo.jar", "cmdline_key": "java:foo",
            "cpu_j": t,  # grows by 60 each step => 1 jiffy/sec => 1% on HZ=100
            "rss_kb": r, "io_r": 0, "io_w": 0,
        })
    df = compute_deltas(_df(rows))
    s = summarize(df)
    row = s.iloc[0]
    assert row["rss_kb_peak"] == 9999
    # p95 of mostly-1000 with 1% outlier should still be 1000-ish, not 9999
    assert row["rss_kb_p95"] < 5000


def test_summarize_separates_co_located_services():
    rows = []
    for t in range(0, 240, 60):
        rows.append({"host": "h", "pid": 1, "ts": t, "comm": "java",
                     "cmdline": "java -jar order.jar", "cmdline_key": "java:order",
                     "cpu_j": t, "rss_kb": 1000, "io_r": 0, "io_w": 0})
        rows.append({"host": "h", "pid": 2, "ts": t, "comm": "java",
                     "cmdline": "java -jar user.jar", "cmdline_key": "java:user",
                     "cpu_j": t, "rss_kb": 2000, "io_r": 0, "io_w": 0})
    df = compute_deltas(_df(rows))
    s = summarize(df)
    assert set(s["key"]) == {"java:order", "java:user"}


def test_top_n_orders_descending():
    rows = []
    keys = [("a", 100), ("b", 500), ("c", 50), ("d", 1000), ("e", 250)]
    for k, rss in keys:
        for t in range(0, 240, 60):
            rows.append({"host": "h", "pid": 1, "ts": t, "comm": k,
                         "cmdline": k, "cmdline_key": k,
                         "cpu_j": t, "rss_kb": rss, "io_r": 0, "io_w": 0})
    df = compute_deltas(_df(rows))
    s = summarize(df)
    top = top_n(s, "h", "rss_kb_peak", n=3)
    assert list(top["key"]) == ["d", "b", "e"]


def test_aggregate_by_service_sums_concurrent_pids():
    # Two celery workers (different pids) running concurrently, same key.
    # At each timestamp their RSS/CPU/IO should sum into a single row.
    rows = []
    for pid, rss_base in [(100, 200), (200, 300)]:
        for t in range(0, 240, 60):
            rows.append({
                "host": "h", "pid": pid, "ts": t, "comm": "celery",
                "cmdline": f"celery worker {pid}", "cmdline_key": "celery@svc",
                "cpu_j": t * pid, "rss_kb": rss_base, "io_r": pid * t * 10,
                "io_w": pid * t * 5,
            })
    df = compute_deltas(_df(rows))
    out = aggregate_by_service(df)

    # 4 timestamps × 1 key = 4 rows after aggregation (was 8 pre-aggregation)
    assert len(out) == 4
    # At each ts the RSS should be 200 + 300 = 500 (sum of two workers)
    assert all(out["rss_kb"] == 500)
    # io_dr at ts=60: pid100 delta = 100*60*10 - 0 = 60000;
    #                  pid200 delta = 200*60*10 - 0 = 120000; sum = 180000
    row_60 = out[out["ts"] == 60].iloc[0]
    assert row_60["io_dr"] == 180000


def test_aggregate_by_service_preserves_single_pid():
    # Single-pid service: aggregation is a no-op (same row count, same values).
    rows = [{
        "host": "h", "pid": 1, "ts": t, "comm": "java",
        "cmdline": "java -jar foo.jar", "cmdline_key": "java:foo",
        "cpu_j": t, "rss_kb": 1000, "io_r": 0, "io_w": 0,
    } for t in range(0, 240, 60)]
    df = compute_deltas(_df(rows))
    out = aggregate_by_service(df)
    assert len(out) == len(df)
    assert all(out["rss_kb"] == 1000)


def test_aggregate_by_service_empty():
    out = aggregate_by_service(pd.DataFrame())
    assert out.empty


def test_trend_pct_growing_memory():
    # 7 days of growth from 1000 to 2000 RSS — trend should be ~+100%
    rows = []
    for i, t in enumerate(range(0, 7 * 24 * 3600, 600)):
        rss = 1000 + (i / 1008) * 1000  # 1008 samples over a week
        rows.append({"host": "h", "pid": 1, "ts": t, "comm": "x",
                     "cmdline": "x", "cmdline_key": "x",
                     "cpu_j": t, "rss_kb": int(rss), "io_r": 0, "io_w": 0})
    df = compute_deltas(_df(rows))
    s = summarize(df)
    trend = s.iloc[0]["rss_trend_pct"]
    assert 50 < trend < 150  # growth should be substantial
