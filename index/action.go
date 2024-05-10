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
			template.ShowShellExecutePage(bs, "解锁Vault", action.UnsealVaultScript)
		},
	},
	{
		Name: "重载Casbin Mesh规则",
		Action: func(bs *model.AppModel) {
			template.ShowShellExecutePage(bs, "重载Casbin Mesh规则", action.ReloadCasbin)
		},
	},
}
