package index

import (
	"weterm/model"
	"weterm/pages/healthcheck"
	"weterm/pages/template"
)

var aioMenu = []MenuItem{
	{
		Name: "IP切换",
		Action: func(bs *model.AppModel) {
			template.ShowShellExecutePage(bs, "一体机服务初始化", "bash /data/install/aio/init_ip.sh 2>&1")
		},
	},
	{
		Name: "服务概览",
		Action: func(bs *model.AppModel) {
			c := healthcheck.NewConsulHealth()
			template.ShowHealthView(bs, c)
		},
	},
}
