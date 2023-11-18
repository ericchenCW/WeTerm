package ui

import (
	"github.com/rivo/tview"
)

func SetUpMenuPage(receiver *BootStrap) {
	// 应用首页
	listMenu := tview.NewList()
	listMenu.AddItem("表单示例", "", 's', func() {
		SetUpFormSamplePage(receiver) // 创建新的表单示例页面
		receiver.CorePages.SwitchToPage("form_sample")
	})

	listMenu.AddItem("安装", "", 'a', func() {
	})

	listMenu.AddItem("部署", "", 'b', func() {
	})

	listMenu.AddItem("组件健康检查", "", 'c', func() {
		SetUpStatusPage(receiver) // 创建新的状态检查页面
		receiver.CorePages.SwitchToPage("status_check")
	})

	listMenu.AddItem("退出", "", 'q', func() {
		receiver.CoreApp.Stop()
	})

	receiver.CorePages.AddPage("main_menu", listMenu, true, true)
}
