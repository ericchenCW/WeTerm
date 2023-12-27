package template

import (
	"fmt"
	"weterm/model"
	"weterm/pages/healthcheck"

	"github.com/rivo/tview"
)

func ShowHealthView(receiver *model.AppModel, h healthcheck.Health) {
	outputTextView := tview.NewTextView().SetTextAlign(tview.AlignLeft).SetDynamicColors(true)
	outputTextView.SetBorder(true).SetTitle("健康检查")
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(outputTextView, 0, 1, true)
	receiver.CorePages.AddPage("health_check", layout, true, false)
	receiver.CorePages.SwitchToPage("health_check")
	go func() {
		w := tview.ANSIWriter(outputTextView)
		for _, row := range h.Check() {
			receiver.CoreApp.QueueUpdateDraw(func() {
				fmt.Fprintln(w, h.Print(row))
				outputTextView.ScrollToEnd()
			})
		}
	}()
}
