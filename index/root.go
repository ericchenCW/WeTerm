package index

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"weterm/model"
	"weterm/pages"
	"weterm/pages/example"
)

type MenuItem struct {
	Name     string
	Action   func(*model.AppModel)
	SubItems []MenuItem
}

// Main menu items
var mainMenuItems = []MenuItem{
	{
		Name: "示例",
		Action: func(bs *model.AppModel) {
			bs.CorePages.SwitchToPage("form_sample")
		},
		SubItems: []MenuItem{
			{
				Name: "基础表单",
				Action: func(bs *model.AppModel) {
					example.SetUpFormSamplePage(bs)
					bs.CorePages.SwitchToPage("form_sample")
				},
			},
			{
				Name: "Shell示例",
				Action: func(bs *model.AppModel) {
					example.SetUpShellCommandPage(bs)
					bs.CorePages.SwitchToPage("shell_command_page")
				},
			},
			{
				Name: "查看日志",
				Action: func(bs *model.AppModel) {
					example.SetUpLogViewerPage(bs)
					bs.CorePages.SwitchToPage("log_viewer_page")
				},
			},
			{
				Name: "文本编辑",
				Action: func(bs *model.AppModel) {
					example.SetupEditFilePage(bs)
					bs.CorePages.SwitchToPage("edit_file_page")
				},
			},
		},
	},
	{
		Name: "WeOps安装",
		Action: func(bs *model.AppModel) {
			// TODO: Add implementation
		},
		SubItems: []MenuItem{
			{
				Name: "单机版",
				Action: func(bs *model.AppModel) {
				},
			},
			{
				Name: "标准版(3节点)",
				Action: func(bs *model.AppModel) {
				},
			},
			{
				Name: "高可用版(7节点)",
				Action: func(bs *model.AppModel) {
				},
			},
		},
	},
	{
		Name: "运维工具",
		Action: func(bs *model.AppModel) {
			// TODO: Add implementation
		},
		SubItems: []MenuItem{
			// TODO: Add sub-menu items for 运维工具
		},
	},
	{
		Name: "健康检查",
		Action: func(bs *model.AppModel) {
			bs.CorePages.SwitchToPage("status_check")
		},
		SubItems: []MenuItem{
			{
				Name: "组件检查",
				Action: func(bs *model.AppModel) {
					pages.SetUpStatusPage(bs)
					bs.CorePages.SwitchToPage("status_check")
				},
			},
		},
	},
	{
		Name: "退出",
		Action: func(bs *model.AppModel) {
			bs.CoreApp.Stop()
		},
	},
}

func SetUpMenuPage(receiver *model.AppModel) {
	// Main Menu
	mainMenu := createMainMenu(receiver)

	// Sub Menu
	subMenu := createSubMenu(receiver)

	updateSubMenu := func(index int, mainText string, secondaryText string, shortcut rune) {
		// Update submenu based on main menu selection
		subMenu.Clear()
		if index >= 0 && index < len(mainMenuItems) {
			subItems := mainMenuItems[index].SubItems
			for _, item := range subItems {
				action := item.Action // Create a new variable to store the action
				subMenu.AddItem(item.Name, "", 0, func() {
					action(receiver) // Use the action variable instead of item.Action
				})
			}
		}
	}

	mainMenu.SetChangedFunc(updateSubMenu)

	// Call the function manually to set the submenu of the first item
	updateSubMenu(0, mainMenuItems[0].Name, "", 0)

	// Define layout
	flex := tview.NewFlex().SetDirection(tview.FlexColumn)
	flex.AddItem(mainMenu, 0, 1, true)
	flex.AddItem(subMenu, 0, 2, false)

	setMenuInputCapture(receiver, mainMenu, subMenu)

	receiver.CorePages.AddPage("menu", flex, true, true)
	receiver.CoreApp.SetRoot(receiver.CorePages, true)
}

func createMainMenu(receiver *model.AppModel) *tview.List {
	mainMenu := tview.NewList()
	for _, item := range mainMenuItems {
		action := item.Action // Create a new variable to store the action
		mainMenu.AddItem(item.Name, "", 0, func() {
			action(receiver) // Use the action variable instead of item.Action
		})
	}
	mainMenu.SetBorder(true).SetTitle("主菜单")
	receiver.CorePages.AddPage("main_menu", mainMenu, true, true)
	return mainMenu
}

func createSubMenu(receiver *model.AppModel) *tview.List {
	subMenu := tview.NewList()
	subMenu.SetBorder(true).SetTitle("子菜单")
	receiver.CorePages.AddPage("sub_menu", subMenu, false, false)
	return subMenu
}

func setMenuInputCapture(receiver *model.AppModel, mainMenu *tview.List, subMenu *tview.List) {
	mainMenu.SetInputCapture(switchFocusFunc(receiver.CoreApp, subMenu, tcell.KeyRight))
	subMenu.SetInputCapture(switchFocusFunc(receiver.CoreApp, mainMenu, tcell.KeyLeft))
}

func switchFocusFunc(app *tview.Application, target *tview.List, key tcell.Key) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == key {
			app.SetFocus(target)
			return nil
		}
		return event
	}
}
