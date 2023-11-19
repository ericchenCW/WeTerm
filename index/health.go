package index

import (
	"weterm/model"
	"weterm/pages"
)

var healthMenu = []MenuItem{
	{
		Name: "组件检查",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
}
