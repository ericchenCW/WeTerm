package template

import (
	"bufio"
	"fmt"
	"github.com/rivo/tview"
	"os/exec"
	"weterm/model"
)

func ShowShellExecutePage(receiver *model.AppModel, title string, shellCommand string) {
	outputTextView := tview.NewTextView().SetDynamicColors(true)
	outputTextView.SetBorder(true).SetTitle(title).SetTitleAlign(tview.AlignCenter)
	outputTextView.SetScrollable(true)

	// 执行命令并获取输出
	cmd := exec.Command("bash", "-c", shellCommand)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(outputTextView, "Error: %v\n", err)
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(outputTextView, "Error: %v\n", err)
		return
	}

	// 创建一个页面
	receiver.CorePages.AddPage("shell_execute_page", outputTextView, true, false)
	receiver.CorePages.SwitchToPage("shell_execute_page")

	// start a goroutine to read the command's output
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			// use the application's queue to update the UI
			receiver.CoreApp.QueueUpdateDraw(func() {
				fmt.Fprintf(outputTextView, "Command Output: %s\n", scanner.Text())
				outputTextView.ScrollToEnd() // Scroll to end
			})
		}
		if err := scanner.Err(); err != nil {
			receiver.CoreApp.QueueUpdateDraw(func() {
				fmt.Fprintf(outputTextView, "Error: %v\n", err)
			})
		}
		if err := cmd.Wait(); err != nil {
			receiver.CoreApp.QueueUpdateDraw(func() {
				fmt.Fprintf(outputTextView, "Error: %v\n", err)
			})
		}
	}()
}
