package index

import (
	"weterm/model"
	"weterm/pages/aio"
	"weterm/pages/template"
)

var aioMenu = []MenuItem{
	{
		Name: "服务初始化—IP更新",
		Action: func(bs *model.AppModel) {
			template.ShowShellExecutePage(bs, "服务初始化", aio.InitIPScript)
		},
	},
	{
		Name: "服务器关机",
		Action: func(bs *model.AppModel) {
			template.ShowShellExecutePage(bs, "服务停止后关机", aio.StopScript)
		},
	},
}
