package index

import (
	"weterm/model"
	procmonpage "weterm/pages/procmon"
)

// procmonMenu 是「进程监控」子菜单，shell-out 编排 procmon 采集器（远端 cron）与
// Python 报告器的生命周期。
var procmonMenu = []MenuItem{
	{
		Name: "部署采集器",
		Action: func(bs *model.AppModel) {
			procmonpage.Deploy(bs)
		},
	},
	{
		Name: "拉取数据",
		Action: func(bs *model.AppModel) {
			procmonpage.Pull(bs)
		},
	},
	{
		Name: "生成报告",
		Action: func(bs *model.AppModel) {
			procmonpage.Report(bs)
		},
	},
	{
		Name: "卸载采集器",
		Action: func(bs *model.AppModel) {
			procmonpage.Uninstall(bs)
		},
	},
}
