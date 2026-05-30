#!/usr/bin/env python3
"""Generate a synthetic week of procmon JSONL data for smoke-testing the reporter.

Usage:  python3 gen_synthetic.py <out-dir>

Writes ``{host}-YYYY-MM-DD.jsonl`` files mimicking the collector output for
4 hosts × 7 days × ~10 processes × 1440 samples per day.
"""
from __future__ import annotations

import datetime as dt
import json
import random
import sys
from pathlib import Path

HOSTS = ["host-prod-a", "host-prod-b", "host-stage", "host-bench"]

# Per-host process catalog: each entry is (comm, cmdline, base_rss_kb,
# rss_growth_rate_per_day_kb, base_cpu_jiffies_per_sample, base_io_per_sample).
HOST_PROCESSES = {
    "host-prod-a": [
        ("java",    "/usr/bin/java -Xmx16g -jar /opt/order-svc.jar --port=9090",   12_000_000,  200_000, 800, 2_000_000),
        ("java",    "/usr/bin/java -Xmx8g -jar /opt/user-svc.jar --port=9091",     6_500_000,   50_000,  400, 800_000),
        ("mysqld",  "/usr/sbin/mysqld --defaults-file=/etc/my.cnf",                 8_388_608,    5_000,  600, 10_000_000),
        ("redis-server", "redis-server *:6379",                                     1_500_000,        0,  300, 100_000),
        ("nginx",   "nginx: worker process",                                            200_000,        0,  100, 5_000_000),
        ("python3", "python3 /opt/cron/billing.py --name billing-cron",                500_000,    1_000,   80, 50_000),
        ("sshd",    "/usr/sbin/sshd -D",                                                10_000,        0,   10, 1_000),
        ("systemd", "/lib/systemd/systemd --system --deserialize 18",                    8_000,        0,    5, 500),
    ],
    "host-prod-b": [
        ("java",    "/usr/bin/java -Xmx16g -jar /opt/order-svc.jar --port=9090",   12_500_000,  220_000, 850, 2_100_000),
        ("java",    "/usr/bin/java -Xmx4g -jar /opt/notify-svc.jar",                 3_500_000,   10_000,  300, 400_000),
        ("mysqld",  "/usr/sbin/mysqld --defaults-file=/etc/my.cnf",                  8_388_608,    3_000,  600, 9_500_000),
        ("nginx",   "nginx: worker process",                                            200_000,        0,  120, 6_000_000),
        ("python3", "python3 /opt/cron/billing.py --name billing-cron",                500_000,      500,   80, 50_000),
    ],
    "host-stage": [
        ("java",    "/usr/bin/java -Xmx2g com.example.TestRunner --suite=full",     1_800_000,        0,  200, 50_000),
        ("postgres", "postgres: 14/main",                                            1_000_000,    1_000,  150, 1_000_000),
        ("python3", "python3 -m worker --name etl-loader",                            800_000,    2_000,  100, 800_000),
    ],
    "host-bench": [
        ("stress",  "stress --cpu 8 --io 4 --vm 2 --vm-bytes 1G --timeout 0",       1_200_000,        0, 7000, 50_000_000),
        ("python3", "python3 bench.py --workload mixed",                              400_000,        0,  500, 5_000_000),
    ],
}


def main(out_dir: Path) -> None:
    out_dir.mkdir(parents=True, exist_ok=True)
    random.seed(42)  # reproducible

    today = dt.date.today()
    days = 7
    sample_interval = 60  # seconds

    for host in HOSTS:
        procs = HOST_PROCESSES[host]
        # pid stable across all samples in our synthetic data
        pids = [1000 + i for i in range(len(procs))]

        for day_offset in range(days):
            day = today - dt.timedelta(days=days - 1 - day_offset)
            day_start = dt.datetime.combine(day, dt.time(0, 0))

            cum_cpu = [0] * len(procs)
            cum_io_r = [0] * len(procs)
            cum_io_w = [0] * len(procs)

            file_path = out_dir / f"{host}-{day.isoformat()}.jsonl"
            with file_path.open("w") as f:
                samples_per_day = 24 * 60  # 1440
                for s in range(samples_per_day):
                    ts = int((day_start + dt.timedelta(seconds=s * sample_interval)).timestamp())
                    uptime_s = day_offset * 86400 + s * sample_interval + 100_000
                    for i, (comm, cmd, base_rss, growth, cpu_per, io_per) in enumerate(procs):
                        # Diurnal pattern: business hours 9-18 are heavier.
                        hour = (s * sample_interval // 3600) % 24
                        load_mul = 1.4 if 9 <= hour <= 18 else 0.5
                        # Noise
                        jitter = random.uniform(0.85, 1.15)
                        cum_cpu[i] += int(cpu_per * load_mul * jitter)
                        cum_io_r[i] += int(io_per * 0.6 * load_mul * jitter)
                        cum_io_w[i] += int(io_per * 0.4 * load_mul * jitter)
                        # RSS grows linearly across the week + small noise.
                        days_in = day_offset + s / samples_per_day
                        rss = int(base_rss + growth * days_in
                                  + random.uniform(-base_rss * 0.01, base_rss * 0.01))
                        # bench host: simulate occasional spikes on the stress process
                        if host == "host-bench" and i == 0 and random.random() < 0.05:
                            rss = int(rss * 2.5)
                        rec = {
                            "ts": ts,
                            "host": host,
                            "pid": pids[i],
                            "comm": comm,
                            "cmdline": cmd,
                            "cpu_j": cum_cpu[i],
                            "rss_kb": rss,
                            "io_r": cum_io_r[i],
                            "io_w": cum_io_w[i],
                            "uptime_s": uptime_s,
                        }
                        f.write(json.dumps(rec) + "\n")
            print(f"wrote {file_path}")

    print(f"done. {len(HOSTS)} hosts × {days} days under {out_dir}")


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("usage: gen_synthetic.py <out-dir>", file=sys.stderr)
        sys.exit(2)
    main(Path(sys.argv[1]))
