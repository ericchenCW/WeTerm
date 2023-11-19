package index

import "weterm/model"

var installMenu = []MenuItem{
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
}
