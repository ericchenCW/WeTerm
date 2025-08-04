package index

import (
	"weterm/model"
	"weterm/pages/action"
	"weterm/pages/template"
)

var componentActionMenu = []MenuItem{
	{
		Name: "解锁Vault",
		Action: func(bs *model.AppModel) {
			actionInfo, _ := action.GetAction("unseal_vault")
			template.ShowShellExecutePage(bs, "解锁Vault", actionInfo.Script)
		},
	},
	{
		Name: "重载Casbin Mesh规则",
		Action: func(bs *model.AppModel) {
			actionInfo, _ := action.GetAction("reload_casbin")
			template.ShowShellExecutePage(bs, "重载Casbin Mesh规则", actionInfo.Script)
		},
	},
	{
		Name: "备份MongoDB",
		Action: func(bs *model.AppModel) {
			actionInfo, _ := action.GetAction("backup_mongodb")
			template.ShowShellExecutePage(bs, "备份MongoDB", actionInfo.Script)
		},
	},
	{
		Name: "清理RabbitMQ队列",
		Action: func(bs *model.AppModel) {
			actionInfo, _ := action.GetAction("purge_rabbitmq")
			template.ShowShellExecutePage(bs, "清理RabbitMQ队列", actionInfo.Script)
		},
	},
	{
		Name: "备份MySQL",
		Action: func(bs *model.AppModel) {
			actionInfo, _ := action.GetAction("backup_mysql")
			template.ShowShellExecutePage(bs, "备份MySQL", actionInfo.Script)
		},
	},
	{
		Name: "发送Trace",
		Action: func(bs *model.AppModel) {
			action.SendTraceView(bs)
		},
	},
	{
		Name: "接收告警",
		Action: func(bs *model.AppModel) {
			action.DebugAlarmView(bs)
		},
	},
}
