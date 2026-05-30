// Package inspect 是 WeTerm「平台巡检」页，in-process 调用 weops-inspect 的
// runner 执行全量巡检，进度实时刷进 TUI，完成后展示汇总并落地 HTML 报告。
package inspect

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"weterm/model"

	"weops-inspect/config"
	"weops-inspect/output"
	"weops-inspect/runner"

	"github.com/rivo/tview"
)

const inspectPageName = "inspect_page"

// ShowInspectPage 触发一次全量平台巡检。进度通过 runner 的 progress 回调实时刷进
// TUI；完成后展示汇总并把 HTML 报告落地到输出目录。ESC 经 AppModel.CancelFunc
// 取消，runner 在阶段间检查 ctx 后尽快返回。
func ShowInspectPage(receiver *model.AppModel) {
	view := tview.NewTextView().SetDynamicColors(true)
	view.SetBorder(true).SetTitle("平台巡检").SetTitleAlign(tview.AlignCenter)
	view.SetScrollable(true)

	receiver.CorePages.AddPage(inspectPageName, view, true, false)
	receiver.CorePages.SwitchToPage(inspectPageName)

	ctx, cancel := context.WithCancel(context.Background())
	receiver.CancelFunc = cancel

	write := func(s string) {
		receiver.CoreApp.QueueUpdateDraw(func() {
			fmt.Fprintln(view, s)
			view.ScrollToEnd()
		})
	}

	go func() {
		// 复用 WeTerm 进程内已 godotenv.Load 的 BK_* 环境变量。
		cfg, err := config.Load(".")
		if err != nil {
			write(fmt.Sprintf("[red]配置加载失败: %v[white]", err))
			return
		}
		if err := cfg.Validate(); err != nil {
			write(fmt.Sprintf("[red]配置校验失败: %v[white]", err))
			return
		}

		write("[yellow]开始全量巡检...[white]")
		report, err := runner.Run(ctx, cfg, write)
		if err != nil {
			write(fmt.Sprintf("[red]巡检失败: %v[white]", err))
			return
		}

		write(fmt.Sprintf("\n[green]巡检完成[white]! 共 %d 项检查, [green]%d 正常[white], [red]%d 告警[white], %d 未知",
			report.Summary.Total, report.Summary.OK, report.Summary.Warn, report.Summary.Unknown))

		htmlPath, err := output.Write(report, cfg.OutputDir)
		if err != nil {
			write(fmt.Sprintf("[yellow]报告写入失败: %v[white]", err))
			return
		}
		write(fmt.Sprintf("报告已生成: [green]%s[white]", htmlPath))
		write("[white]按 ESC 返回主菜单")
	}()
}

// ShowLatestReport 显示输出目录下最近一次巡检报告的绝对路径，供用户用浏览器或
// scp 查看（一体机 TUI 环境不直接拉起浏览器）。
func ShowLatestReport(receiver *model.AppModel) {
	view := tview.NewTextView().SetDynamicColors(true)
	view.SetBorder(true).SetTitle("最近巡检报告").SetTitleAlign(tview.AlignCenter)
	view.SetScrollable(true)
	receiver.CorePages.AddPage(inspectPageName, view, true, false)
	receiver.CorePages.SwitchToPage(inspectPageName)

	dir := "."
	if cfg, err := config.Load("."); err == nil {
		dir = cfg.OutputDir
	}

	matches, _ := filepath.Glob(filepath.Join(dir, "weops_inspection*.html"))
	if len(matches) == 0 {
		fmt.Fprintln(view, "[yellow]未找到巡检报告，请先执行全量巡检。[white]")
		return
	}

	// 按修改时间取最新一份。
	sort.Slice(matches, func(i, j int) bool {
		fi, errI := os.Stat(matches[i])
		fj, errJ := os.Stat(matches[j])
		if errI != nil || errJ != nil {
			return false
		}
		return fi.ModTime().After(fj.ModTime())
	})

	abs, err := filepath.Abs(matches[0])
	if err != nil {
		abs = matches[0]
	}
	fmt.Fprintf(view, "最近报告: [green]%s[white]\n", abs)
	fmt.Fprintln(view, "可用浏览器打开，或 scp 到本地查看。")
}
