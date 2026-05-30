package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"weops-inspect/config"
	"weops-inspect/lock"
	"weops-inspect/notify"
	"weops-inspect/output"
	"weops-inspect/runner"
)

var version = "dev"

func main() {
	outputDir := flag.String("o", ".", "输出目录")
	showVersionShort := flag.Bool("v", false, "打印版本号并退出")
	showVersionLong := flag.Bool("version", false, "打印版本号并退出")
	flag.Parse()

	if *showVersionShort || *showVersionLong {
		fmt.Println(version)
		return
	}

	// Load config from BK_* environment variables
	cfg, err := config.Load(*outputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置加载失败: %v\n", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "配置校验失败: %v\n", err)
		os.Exit(1)
	}

	// Single-instance guard. Lock lives next to notify state.json so it
	// shares the user-private config dir. If another instance holds the
	// lock, exit code 0 — overlapping cron triggers are protective skips,
	// not failures (a non-zero exit would make cron mail the operator).
	if notifyPath, err := notify.ConfigPath(); err == nil {
		lockPath := filepath.Join(filepath.Dir(notifyPath), "inspect.lock")
		release, err := lock.Acquire(lockPath)
		switch {
		case err == nil:
			defer release()
		case errors.Is(err, lock.ErrBusy):
			fmt.Fprintln(os.Stderr, "weops-inspect: another instance is running, exiting")
			return
		default:
			fmt.Fprintf(os.Stderr, "weops-inspect: lock unavailable, continuing without it: %v\n", err)
		}
	}

	// 三阶段采集 + 规则判定 + 汇总（进度打到 stderr，与原行为一致）。
	progress := func(s string) { fmt.Fprintln(os.Stderr, s) }
	report, err := runner.Run(context.Background(), cfg, progress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "巡检失败: %v\n", err)
		os.Exit(1)
	}

	// Optional notify config: when enabled, persistence confirmation runs
	// BEFORE rendering so the on-disk HTML/JSON match what we notify on.
	// Pending warns are demoted to Notice (excluded from Summary).
	notifyCfg, err := notify.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "notify: 配置加载失败: %v\n", err)
		notifyCfg = nil
	}
	prep := notify.Prepare(notifyCfg, report)

	// Output (after persistence demotion so HTML, JSON, and Summary agree)
	fmt.Fprintf(os.Stderr, "\n生成报告...\n")
	htmlPath, err := output.Write(report, cfg.OutputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "报告生成失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "\n巡检完成! 共 %d 项检查, %d 正常, %d 告警, %d 未知\n",
		report.Summary.Total, report.Summary.OK, report.Summary.Warn, report.Summary.Unknown)

	notify.Dispatch(prep, notifyCfg, report, htmlPath)
}
