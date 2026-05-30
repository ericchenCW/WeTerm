package inspect

import (
	"context"
	"fmt"

	"weterm/model"

	"weops-inspect/config"
	imodel "weops-inspect/model"
	"weops-inspect/runner"

	"github.com/rivo/tview"
)

const overviewPageName = "overview_page"

// statusColor 把 inspect 的检查状态映射为 tview 颜色标签。
func statusColor(s imodel.CheckStatus) string {
	switch s {
	case imodel.StatusOK:
		return "green"
	case imodel.StatusWarn:
		return "red"
	case imodel.StatusUnknown:
		return "yellow"
	default:
		return "white"
	}
}

// ShowOverviewPage 是「服务概览」轻量速查：只跑 inspect 的主机指标采集与判定
// （RunHostsOnly），在 TUI 内逐项展示主机健康态，不生成 HTML 报告、不探测开源
// 组件，用于快速查看而非全量巡检。
func ShowOverviewPage(receiver *model.AppModel) {
	view := tview.NewTextView().SetDynamicColors(true)
	view.SetBorder(true).SetTitle("服务概览（主机速查）").SetTitleAlign(tview.AlignCenter)
	view.SetScrollable(true)

	receiver.CorePages.AddPage(overviewPageName, view, true, false)
	receiver.CorePages.SwitchToPage(overviewPageName)

	ctx, cancel := context.WithCancel(context.Background())
	receiver.CancelFunc = cancel

	write := func(s string) {
		receiver.CoreApp.QueueUpdateDraw(func() {
			fmt.Fprintln(view, s)
			view.ScrollToEnd()
		})
	}

	go func() {
		cfg, err := config.Load(".")
		if err != nil {
			write(fmt.Sprintf("[red]配置加载失败: %v[white]", err))
			return
		}
		if err := cfg.Validate(); err != nil {
			write(fmt.Sprintf("[red]配置校验失败: %v[white]", err))
			return
		}

		write("[yellow]采集主机健康态...[white]")
		report, err := runner.RunHostsOnly(ctx, cfg, write)
		if err != nil {
			write(fmt.Sprintf("[red]采集失败: %v[white]", err))
			return
		}

		write("")
		for _, h := range report.Hosts {
			write(fmt.Sprintf("[white]主机 [aqua]%s[white]:", h.Metrics.IP))
			if h.Metrics.Error != "" {
				write(fmt.Sprintf("  [red]不可达: %s[white]", h.Metrics.Error))
				continue
			}
			for _, c := range h.Checks {
				color := statusColor(c.Status)
				line := fmt.Sprintf("  [%s]%-20s %s[white]", color, c.Field, c.Value)
				if c.Threshold != "" {
					line += fmt.Sprintf(" (阈值 %s)", c.Threshold)
				}
				write(line)
			}
		}

		s := report.Summary
		write(fmt.Sprintf("\n[white]汇总: 共 %d 项, [green]%d 正常[white], [red]%d 告警[white], [yellow]%d 未知[white]",
			s.Total, s.OK, s.Warn, s.Unknown))
		write("[white]按 ESC 返回主菜单")
	}()
}
