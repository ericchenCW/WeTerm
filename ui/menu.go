package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MenuItem struct {
	Name   string
	Action func(*BootStrap)
}

// Main menu items
var (
	mainMenuItems = []MenuItem{
		{"示例", func(bs *BootStrap) {
			bs.CorePages.SwitchToPage("form_sample")
		}},
		{"WeOps安装", func(bs *BootStrap) {
		}},
		{"运维工具", func(bs *BootStrap) {
		}},
		{"健康检查", func(bs *BootStrap) {
			bs.CorePages.SwitchToPage("status_check")
		}},

		{"退出", func(bs *BootStrap) {
			bs.CoreApp.Stop()
		}},
	}
)

// Sub menu items
var (
	subMenuItems = map[string]func(*BootStrap){
		"基础表单":    SetUpFormSamplePage,
		"组件检查":    SetUpStatusPage,
		"Shell示例": SetUpShellCommandPage,
	}
)

func SetUpMenuPage(receiver *BootStrap) {
	// Main Menu
	mainMenu := createMainMenu(receiver)

	// Sub Menu
	subMenu := createSubMenu(receiver)

	updateSubMenu := func(index int, mainText string, secondaryText string, shortcut rune) {
		// Update submenu based on main menu selection
		subMenu.Clear()
		switch mainText {
		case "示例":
			subMenu.AddItem("基础表单", "", 0, func() {
				subMenuItems["基础表单"](receiver)
				receiver.CorePages.SwitchToPage("form_sample")
			})
			subMenu.AddItem("Shell示例", "", 0, func() { // 添加这段代码
				subMenuItems["Shell示例"](receiver)
				receiver.CorePages.SwitchToPage("shell_command_page")
			})
		case "WeOps安装":
			subMenu.AddItem("单机版", "", 0, nil)
			subMenu.AddItem("标准版(3节点)", "", 0, nil)
			subMenu.AddItem("高可用版(7节点)", "", 0, nil)
		case "健康检查":
			subMenu.AddItem("组件检查", "", 0, func() {
				subMenuItems["组件检查"](receiver)
				receiver.CorePages.SwitchToPage("status_check")
			})
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

func createMainMenu(receiver *BootStrap) *tview.List {
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

func createSubMenu(receiver *BootStrap) *tview.List {
	subMenu := tview.NewList()
	subMenu.SetBorder(true).SetTitle("子菜单")
	receiver.CorePages.AddPage("sub_menu", subMenu, false, false)
	return subMenu
}

func setMenuInputCapture(receiver *BootStrap, mainMenu *tview.List, subMenu *tview.List) {
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
