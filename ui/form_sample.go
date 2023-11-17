package ui

import (
	"context"
	"fmt"
	"github.com/rivo/tview"
	"os/exec"
	"strings"
	"time"
)

func SetUpFormSamplePage(receiver *BootStrap) {
	formPage := tview.NewForm()
	formPage.AddTextArea("APPO服务器", "", 20, 0, 0, nil)
	formPage.AddTextArea("APPT服务器", "", 20, 0, 0, nil)
	formPage.SetBorder(true).SetTitle("WeOps部署").SetTitleAlign(tview.AlignLeft)

	outputTextView := tview.NewTextView().SetDynamicColors(true)
	outputTextView.SetBorder(true).SetTitle("输出")
	outputTextView.SetText("")

	formPage.AddButton("开始部署", func() {
		ctx, cancel := context.WithCancel(context.Background())
		receiver.CancelFunc = cancel

		outputTextView.SetText("") // 清空输出框的内容
		appoServer := formPage.GetFormItemByLabel("APPO服务器").(*tview.TextArea).GetText()
		apptServer := formPage.GetFormItemByLabel("APPT服务器").(*tview.TextArea).GetText()

		// 输出服务器地址
		fmt.Fprintf(outputTextView, "APPO服务器地址: %s\n", appoServer)
		fmt.Fprintf(outputTextView, "APPT服务器地址: %s\n", apptServer)
		fmt.Fprintf(outputTextView, "开始部署......:")

		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					out, err := exec.Command("sh", "-c", "ps -ef | wc -l").Output()
					if err != nil {
						fmt.Fprintf(outputTextView, "Error: %v\n", err)
					} else {
						fmt.Fprintf(outputTextView, "当前进程数量: %s\n", strings.TrimSpace(string(out)))
					}
					outputTextView.ScrollToEnd()
					receiver.CoreApp.Draw()
					time.Sleep(time.Second)
				}
			}
		}(ctx)
	})

	flex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(formPage, 0, 1, true).
		AddItem(outputTextView, 0, 3, false)

	receiver.CorePages.AddPage("form_sample", flex, true, false)

}
