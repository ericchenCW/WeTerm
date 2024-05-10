package index

import (
	"os"
	"weterm/model"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MenuItem struct {
	Name     string
	Action   func(*model.AppModel)
	SubItems []MenuItem
}

var aioMainMenuItems = []MenuItem{
	{
		Name: "WeOps一体机",
		Action: func(bs *model.AppModel) {
		},
		SubItems: aioMenu,
	},
}

// Main menu items
var mainMenuItems = []MenuItem{
	{
		Name: "服务概览",
		Action: func(bs *model.AppModel) {
		},
		SubItems: componentHealthMenu,
	},
	{
		Name:     "信息收集",
		Action:   func(am *model.AppModel) {},
		SubItems: collectMenu,
	},
	{
		Name:     "配置管理",
		Action:   func(bs *model.AppModel) {},
		SubItems: configMenu,
	},
	{
		Name:     "常用操作",
		Action:   func(bs *model.AppModel) {},
		SubItems: componentActionMenu,
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
	if os.Getenv("AIO") == "true" {
		mainMenuItems = append(aioMainMenuItems, mainMenuItems...)
	}
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
