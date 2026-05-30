// Package runner exposes weops-inspect 的巡检流程作为可被库调用的入口。
//
// 它把原 main.go 中的三阶段采集（主机指标 → 蓝鲸模块 → 开源组件）、规则判定与
// 汇总抽出为 Run，使 WeTerm 等调用方能 in-process 触发巡检并拿到结构化报告，
// 同时 main.go 仍作为薄壳保留独立 CLI 能力。
//
// Run 不负责写盘（output.Write）与告警（notify），这些由调用方按需编排——
// CLI 走 main.go 的原有顺序，WeTerm 巡检页只展示 + 落地 HTML、不发告警。
package runner

import (
	"context"
	"fmt"
	"time"

	"weops-inspect/checker"
	"weops-inspect/collector"
	"weops-inspect/config"
	"weops-inspect/model"
	sshclient "weops-inspect/ssh"
)

// Run 执行一次全量巡检并返回结构化报告。
//
// progress 用于上报阶段进度（如 "[1/3] 采集主机指标..."），为 nil 时静默。
// ctx 用于取消：在各阶段之间检查，取消时尽快返回 ctx.Err()。
func Run(ctx context.Context, cfg *config.Config, progress func(string)) (*model.InspectReport, error) {
	emit := func(format string, args ...interface{}) {
		if progress != nil {
			progress(fmt.Sprintf(format, args...))
		}
	}

	report := &model.InspectReport{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Services:  make(map[string][]model.ServiceStatus),
	}

	// 初始化 SSH 客户端
	sshClient, err := sshclient.New(cfg.SSHUser, cfg.SSHPort, cfg.SSHKeyPath, cfg.SSHUseSudo,
		30*time.Second, 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("SSH 客户端初始化失败: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Phase 1: 采集主机指标
	emit("[1/3] 采集主机指标 (%d 台主机)...", len(cfg.AllHosts))
	hostMetrics := collector.CollectAllHosts(sshClient, cfg.AllHosts, cfg.CheckMountPath, cfg.DiskIncludeNFS)

	var allChecks []model.CheckResult
	for _, hm := range hostMetrics {
		checks := checker.CheckHost(hm, cfg.Thresholds)
		allChecks = append(allChecks, checks...)
		report.Hosts = append(report.Hosts, model.HostCheckResult{
			Metrics: hm,
			Checks:  checks,
		})
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Phase 2: 采集蓝鲸模块状态
	emit("[2/3] 采集蓝鲸模块状态...")
	serviceResults := collector.CollectAllServices(sshClient, cfg)
	report.Services = serviceResults

	// 服务状态规则判定。按 index 迭代，使回填的 RenderStatus /
	// HealthzRenderStatus / ExitedRenderStatus 在模板渲染时保留。
	for moduleKey, statuses := range serviceResults {
		for i := range statuses {
			s := &statuses[i]
			allChecks = append(allChecks, checker.CheckServiceCollectError(s)...)
			for j := range s.Services {
				allChecks = append(allChecks, checker.CheckService(&s.Services[j], s.HostIP, moduleKey)...)
			}
			allChecks = append(allChecks, checker.CheckServiceContainers(s, cfg.Thresholds)...)
		}
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Phase 3: 采集开源组件状态
	emit("[3/3] 采集开源组件状态...")
	report.ES = collector.CollectES(ctx, cfg)
	report.MySQL = collector.CollectMySQL(ctx, cfg)
	report.RedisStandalone = collector.CollectRedisStandalone(ctx, cfg)
	report.RedisSentinel = collector.CollectRedisSentinel(ctx, cfg)
	collector.CrossCheckSentinelMaster(report.RedisSentinel, cfg.RedisMasterIPs)
	report.MongoDB = collector.CollectMongo(ctx, cfg)
	report.RabbitMQ = collector.CollectRabbitMQ(ctx, cfg)
	report.Replication = collector.CollectReplication(ctx, cfg)
	if deps := collector.CollectBKMonitorV3Deps(cfg); deps != nil {
		report.BKMonitorV3 = &model.BKMonitorV3Section{Dependencies: deps}
	}

	// 组件级检查（各 Check* 就地回填 render 状态，并返回 CheckResult 供汇总/告警）。
	allChecks = append(allChecks, checker.CheckES(report.ES, cfg.Thresholds)...)
	allChecks = append(allChecks, checker.CheckRedis(report.RedisStandalone, cfg.Thresholds)...)
	allChecks = append(allChecks, checker.CheckRedisSentinel(report.RedisSentinel)...)
	allChecks = append(allChecks, checker.CheckMongo(report.MongoDB)...)
	allChecks = append(allChecks, checker.CheckRabbitMQ(report.RabbitMQ, cfg.Thresholds)...)
	allChecks = append(allChecks, checker.CheckBKDeps(report.BKMonitorV3)...)
	allChecks = append(allChecks, checker.CheckReplication(report.Replication, cfg.Thresholds)...)

	// 汇总
	report.Summary = checker.Summarize(allChecks)
	report.AllChecks = allChecks

	return report, nil
}
