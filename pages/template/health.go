package template

import (
	"fmt"
	"weterm/model"
	"weterm/pages/healthcheck"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func ShowHealthView(receiver *model.AppModel, h healthcheck.Health) {
	outputTextView := tview.NewTextView().SetTextAlign(tview.AlignLeft).SetDynamicColors(true)
	outputTextView.SetBorder(true).SetTitle("健康检查-按F5刷新")
	outputTextView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF5:
			outputTextView.Clear()
			go Load(receiver, h, outputTextView)
		default:
			return event
		}
		return nil
	})
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(outputTextView, 0, 1, true)
	receiver.CorePages.AddPage("health_check", layout, true, false)
	receiver.CorePages.SwitchToPage("health_check")
	go Load(receiver, h, outputTextView)
}

func Load(receiver *model.AppModel, h healthcheck.Health, outputTextView *tview.TextView) {
	w := tview.ANSIWriter(outputTextView)
	for _, row := range h.Check() {
		receiver.CoreApp.QueueUpdateDraw(func() {
			fmt.Fprintln(w, h.Print(row))
			outputTextView.ScrollToEnd()
		})
	}
}
