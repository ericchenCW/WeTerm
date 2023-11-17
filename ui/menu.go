package ui

import (
	"github.com/rivo/tview"
)

func SetUpMenuPage(receiver BootStrap) {
	// 应用首页
	listMenu := tview.NewList()
	listMenu.AddItem("检查WeOps组件状态", "", 'a', func() {
		receiver.CorePages.SwitchToPage("status_check")
	})

	listMenu.AddItem("退出", "", 'q', func() {
		receiver.CoreApp.Stop()
	})

	receiver.CorePages.AddPage("main_menu", listMenu, true, true)
}
