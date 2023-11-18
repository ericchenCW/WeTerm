package ui

import (
	"context"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type BootStrap struct {
	CoreApp    *tview.Application
	CorePages  *tview.Pages
	CoreList   *tview.List
	CancelFunc context.CancelFunc
}

func NewBootStrap() *BootStrap {
	coreApp := tview.NewApplication()
	corePage := tview.NewPages()
	coreList := tview.NewList()
	coreApp.SetRoot(corePage, true)
	return &BootStrap{CoreApp: coreApp, CorePages: corePage, CoreList: coreList}
}

func (receiver *BootStrap) setupPages() {
	SetUpMenuPage(receiver)
	SetUpStatusPage(receiver)
	SetUpFormSamplePage(receiver)
}

func (receiver *BootStrap) SetupInputCapture() {
	receiver.CoreApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {

		if event.Key() == tcell.KeyEscape {
			if receiver.CancelFunc != nil {
				receiver.CancelFunc()
			}
			receiver.CorePages.SwitchToPage("menu")
			return nil
		}
		return event
	})
}

func (receiver BootStrap) Start() {
	receiver.setupPages()
	receiver.SetupInputCapture()

	if err := receiver.CoreApp.Run(); err != nil {
		panic(err)
	}
}
