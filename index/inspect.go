package index

import (
	"weterm/model"
	inspectpage "weterm/pages/inspect"
)

// inspectMenu 是「平台巡检」子菜单，in-process 调用 weops-inspect 的全量巡检。
var inspectMenu = []MenuItem{
	{
		Name: "执行全量巡检",
		Action: func(bs *model.AppModel) {
			inspectpage.ShowInspectPage(bs)
		},
	},
	{
		Name: "打开最近报告",
		Action: func(bs *model.AppModel) {
			inspectpage.ShowLatestReport(bs)
		},
	},
}
