package template

import (
	"bufio"
	"bytes"
	"fmt"
	"weterm/model"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

func ShowTextViewPage(receiver *model.AppModel, title string, raw bytes.Buffer, doneFunc *func(key tcell.Key)) {
	outputTextView := tview.NewTextView().SetDynamicColors(true)
	outputTextView.SetBorder(true).SetTitle(title).SetTitleAlign(tview.AlignCenter)
	outputTextView.SetScrollable(true)
	if doneFunc != nil {
		log.Logger.Debug().Msg("Set TextView DoneFunction")
		outputTextView.SetDoneFunc(*doneFunc)
	}

	// 创建一个页面
	receiver.CorePages.AddPage("shell_execute_page", outputTextView, true, false)
	receiver.CorePages.SwitchToPage("shell_execute_page")
	receiver.CoreApp.SetFocus(outputTextView)
	// start a goroutine to read the command's output
	go func() {
		w := tview.ANSIWriter(outputTextView)
		scanner := bufio.NewScanner(&raw)
		for scanner.Scan() {
			receiver.CoreApp.QueueUpdateDraw(func() {
				fmt.Fprintln(w, scanner.Text())
			})
		}
	}()
}
