package index

import (
	"weterm/model"
	"weterm/pages/template"
)

var aioMenu = []MenuItem{
	{
		Name: "服务初始化—IP更新",
		Action: func(bs *model.AppModel) {
			template.ShowShellExecutePage(bs, "服务初始化", "bash /data/install/aio/init_ip.sh 2>&1")
		},
	},
	{
		Name: "服务器关机",
		Action: func(bs *model.AppModel) {
			template.ShowShellExecutePage(bs, "服务停止后关机", "bash -x /data/install/aio/stop/stop.sh 2>&1; shutdown 2>&1")
		},
	},
}
