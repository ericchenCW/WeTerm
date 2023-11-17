package ui

import (
	"github.com/rivo/tview"
	"os/exec"
)

func SetUpStatusPage(receiver *BootStrap) {
	cmd := exec.Command("ps", "-ef")
	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	// 创建一个文本框用于显示命令执行结果
	outputTextView := tview.NewTextView().SetText(string(output)).SetTextAlign(tview.AlignLeft).SetDynamicColors(true)

	// 创建一个布局，并将文本框添加到其中
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(outputTextView, 0, 1, true)

	// 创建一个页面
	receiver.CorePages.AddPage("status_check", layout, true, false)
}
