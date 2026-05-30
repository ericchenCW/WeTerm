// Package procmon 是 WeTerm「进程监控」页，编排 procmon 采集器（远端 cron 常驻）
// 与 Python 报告器的生命周期。
//
// procmon 的运行模型决定它无法进 WeTerm 进程：采集器要常驻每台目标机跑 cron，
// 报告器是 Python。因此本页一律走 shell-out——用本地 ssh/scp 分发与回收数据、
// 用本地 python3 生成报告。采集器二进制以 //go:embed 嵌入 WeTerm，部署时落地为
// 临时文件再 scp 到目标机（与 WeTerm 现有 embed 资源套路一致）。
//
// 这里不复用 utils.RunSSH：它把命令包进 bash -c '...'（含单引号的脚本会被破坏）
// 且吞掉错误只返回 stdout。shell-out 到本地 ssh/scp 能拿到真实退出码与 stderr，
// 满足设计的逐步错误处理要求，也直接复用 procmon 自带 deploy.sh 的成熟逻辑。
package procmon

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"weterm/model"

	"github.com/rivo/tview"
)

//go:embed assets/procmon-linux-amd64
var procmonBin []byte

const (
	pageName   = "procmon_page"
	remoteBin  = "/usr/local/bin/procmon"
	remoteData = "/var/log/procmon"
	cronFile   = "/etc/cron.d/procmon"
)

// hosts 从 BK_NODES_IP_COMMA 读取目标机列表（与 inspect 同一套主机来源）。
func hosts() []string {
	raw := os.Getenv("BK_NODES_IP_COMMA")
	var out []string
	for _, h := range strings.Split(raw, ",") {
		if h = strings.TrimSpace(h); h != "" {
			out = append(out, h)
		}
	}
	return out
}

// sshUser 远端登录用户，默认 root（与 utils SSH 约定一致），可用 PROCMON_SSH_USER 覆盖。
func sshUser() string {
	if u := os.Getenv("PROCMON_SSH_USER"); u != "" {
		return u
	}
	return "root"
}

// localDataDir 本地存放回收的 jsonl 的目录，默认 ./procmon-data，可用 PROCMON_DATA_DIR 覆盖。
func localDataDir() string {
	if d := os.Getenv("PROCMON_DATA_DIR"); d != "" {
		return d
	}
	return "procmon-data"
}

// reporterDir Python 报告器源码目录，默认 ./procmon/reporter，可用 PROCMON_REPORTER_DIR 覆盖。
func reporterDir() string {
	if d := os.Getenv("PROCMON_REPORTER_DIR"); d != "" {
		return d
	}
	return filepath.Join("procmon", "reporter")
}

func sshOpts() []string {
	return []string{"-o", "StrictHostKeyChecking=no", "-o", "BatchMode=yes"}
}

// newPage 创建一个流式输出页并返回 view、ctx 与写函数。ESC 经 AppModel.CancelFunc
// 取消，正在执行的 ssh/scp/python 子进程随 ctx 终止。
func newPage(receiver *model.AppModel, title string) (*tview.TextView, context.Context, func(string)) {
	view := tview.NewTextView().SetDynamicColors(true)
	view.SetBorder(true).SetTitle(title).SetTitleAlign(tview.AlignCenter)
	view.SetScrollable(true)
	receiver.CorePages.AddPage(pageName, view, true, false)
	receiver.CorePages.SwitchToPage(pageName)

	ctx, cancel := context.WithCancel(context.Background())
	receiver.CancelFunc = cancel

	write := func(s string) {
		receiver.CoreApp.QueueUpdateDraw(func() {
			fmt.Fprintln(view, s)
			view.ScrollToEnd()
		})
	}
	return view, ctx, write
}

// runStreaming 执行一个命令，把 stdout+stderr 合并流式写入 TUI，返回退出错误。
// dir 为工作目录（report 需在 reporter 目录下跑），传空字符串则用当前目录。
func runStreaming(ctx context.Context, write func(string), dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout // 合并 stderr 到 stdout 管道
	if err := cmd.Start(); err != nil {
		return err
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		write("  " + scanner.Text())
	}
	return cmd.Wait()
}

// Deploy 把嵌入的采集器分发到所有目标机：scp 二进制 + 写 cron + 冒烟采集一次。
func Deploy(receiver *model.AppModel) {
	_, ctx, write := newPage(receiver, "进程监控 — 部署采集器")
	go func() {
		hs := hosts()
		if len(hs) == 0 {
			write("[red]未配置目标机：请设置 BK_NODES_IP_COMMA 环境变量。[white]")
			return
		}

		// 嵌入二进制落地为临时文件供 scp。
		tmp, err := os.CreateTemp("", "procmon-*")
		if err != nil {
			write(fmt.Sprintf("[red]创建临时文件失败: %v[white]", err))
			return
		}
		defer os.Remove(tmp.Name())
		if _, err := tmp.Write(procmonBin); err != nil {
			write(fmt.Sprintf("[red]写入临时二进制失败: %v[white]", err))
			return
		}
		tmp.Close()
		os.Chmod(tmp.Name(), 0755)

		// cron 内容用 base64 传输，规避远端 shell 的引号转义问题。
		cronContent := fmt.Sprintf(`SHELL=/bin/sh
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
* * * * * root %s collect --data-dir %s
0 3 * * * root %s prune --data-dir %s --keep-days 7
`, remoteBin, remoteData, remoteBin, remoteData)
		cronB64 := base64.StdEncoding.EncodeToString([]byte(cronContent))

		user := sshUser()
		for _, h := range hs {
			target := user + "@" + h
			write(fmt.Sprintf("[yellow]==> %s[white]", target))

			scpArgs := append(sshOpts(), tmp.Name(), target+":/tmp/procmon.new")
			if err := runStreaming(ctx, write, "", "scp", scpArgs...); err != nil {
				write(fmt.Sprintf("  [red]scp 失败: %v[white]", err))
				continue
			}

			remoteCmd := fmt.Sprintf(
				"set -e; install -m 0755 /tmp/procmon.new %s; rm -f /tmp/procmon.new; mkdir -p %s; echo %s | base64 -d > %s; chmod 0644 %s; %s collect --data-dir %s; ls -la %s",
				remoteBin, remoteData, cronB64, cronFile, cronFile, remoteBin, remoteData, remoteData)
			sshArgs := append(sshOpts(), target, remoteCmd)
			if err := runStreaming(ctx, write, "", "ssh", sshArgs...); err != nil {
				write(fmt.Sprintf("  [red]远端安装失败: %v[white]", err))
				continue
			}
			write(fmt.Sprintf("  [green]%s 部署完成[white]", target))
		}
		write("\n[white]全部目标机处理完毕。按 ESC 返回主菜单。")
	}()
}

// Pull 从各目标机把 jsonl 数据 scp 回本地数据目录。
func Pull(receiver *model.AppModel) {
	_, ctx, write := newPage(receiver, "进程监控 — 拉取数据")
	go func() {
		hs := hosts()
		if len(hs) == 0 {
			write("[red]未配置目标机：请设置 BK_NODES_IP_COMMA 环境变量。[white]")
			return
		}
		dataDir := localDataDir()
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			write(fmt.Sprintf("[red]创建本地数据目录失败: %v[white]", err))
			return
		}

		user := sshUser()
		for _, h := range hs {
			target := user + "@" + h
			write(fmt.Sprintf("[yellow]==> %s[white]", target))
			// scp 远端 glob 由远端 shell 展开；无匹配时该主机报错，跳过继续。
			scpArgs := append(sshOpts(), target+":"+remoteData+"/*.jsonl", dataDir+"/")
			if err := runStreaming(ctx, write, "", "scp", scpArgs...); err != nil {
				write(fmt.Sprintf("  [yellow]%s 无数据或拉取失败: %v[white]", target, err))
				continue
			}
			write(fmt.Sprintf("  [green]%s 数据已拉取[white]", target))
		}
		write(fmt.Sprintf("\n[white]数据已汇集到 [green]%s[white]。按 ESC 返回主菜单。", dataDir))
	}()
}

// Report 预检 python3 与依赖后，shell-out 调 Python 报告器生成 HTML 报告。
func Report(receiver *model.AppModel) {
	_, ctx, write := newPage(receiver, "进程监控 — 生成报告")
	go func() {
		// 预检 python3 是否存在。
		if _, err := exec.LookPath("python3"); err != nil {
			write("[red]未找到 python3：请先安装 Python 3。[white]")
			return
		}
		// 预检报告器依赖。
		if err := runStreaming(ctx, write, "", "python3", "-c", "import pandas, matplotlib, jinja2"); err != nil {
			write("[red]缺少报告器依赖。请安装后重试：[white]")
			write(fmt.Sprintf("  [yellow]python3 -m pip install -r %s/requirements.txt[white]", reporterDir()))
			return
		}

		repDir := reporterDir()
		if _, err := os.Stat(repDir); err != nil {
			write(fmt.Sprintf("[red]报告器目录不存在: %s[white]", repDir))
			write("[yellow]可用 PROCMON_REPORTER_DIR 指定 reporter 源码目录。[white]")
			return
		}

		dataDir, err := filepath.Abs(localDataDir())
		if err != nil {
			dataDir = localDataDir()
		}
		out := filepath.Join(dataDir, "report.html")

		write("[yellow]生成报告中...[white]")
		// cwd 设为 reporter 目录，使 `python3 -m procmon_report` 能导入到该包。
		err = runStreaming(ctx, write, repDir, "python3", "-m", "procmon_report",
			"--data-dir", dataDir, "--out", out)
		if err != nil {
			write(fmt.Sprintf("[red]报告生成失败: %v[white]", err))
			return
		}
		write(fmt.Sprintf("[green]报告已生成: %s[white]", out))
		write("[white]可用浏览器打开，或 scp 到本地查看。按 ESC 返回主菜单。")
	}()
}

// Uninstall 从各目标机移除 cron 与采集器二进制（保留已采集的数据）。
func Uninstall(receiver *model.AppModel) {
	_, ctx, write := newPage(receiver, "进程监控 — 卸载采集器")
	go func() {
		hs := hosts()
		if len(hs) == 0 {
			write("[red]未配置目标机：请设置 BK_NODES_IP_COMMA 环境变量。[white]")
			return
		}
		user := sshUser()
		for _, h := range hs {
			target := user + "@" + h
			write(fmt.Sprintf("[yellow]==> %s[white]", target))
			remoteCmd := fmt.Sprintf("rm -f %s %s; echo removed (data under %s kept)", cronFile, remoteBin, remoteData)
			sshArgs := append(sshOpts(), target, remoteCmd)
			if err := runStreaming(ctx, write, "", "ssh", sshArgs...); err != nil {
				write(fmt.Sprintf("  [red]%s 卸载失败: %v[white]", target, err))
				continue
			}
			write(fmt.Sprintf("  [green]%s 已卸载[white]", target))
		}
		write("\n[white]卸载完毕（保留远端数据）。按 ESC 返回主菜单。")
	}()
}
