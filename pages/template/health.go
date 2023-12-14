package template

import (
	"fmt"
	"weterm/model"
	"weterm/pages/healthcheck"

	"github.com/rivo/tview"
)

func ShowHealthView(receiver *model.AppModel, h healthcheck.Health) {
	// 创建一个文本框用于显示ulimit数量和emoji表示
	outputTextView := tview.NewTextView().SetTextAlign(tview.AlignLeft).SetDynamicColors(true)
	// 创建一个布局，并将文本框添加到其中
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(outputTextView, 0, 1, true)

	// 创建一个页面
	receiver.CorePages.AddPage("health_check", layout, true, false)
	receiver.CorePages.SwitchToPage("health_check")
	go func() {
		w := tview.ANSIWriter(outputTextView)
		for _, i := range h.Check() {
			receiver.CoreApp.QueueUpdateDraw(func() {
				fmt.Fprintln(w, h.Print(i))
				outputTextView.ScrollToEnd()
			})
		}
	}()
}
