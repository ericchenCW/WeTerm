package index

import (
	"bytes"
	"weterm/model"
	"weterm/pages/config"
	"weterm/pages/template"
)

var configMenu = []MenuItem{
	{
		Name: "覆盖代理配置",
		Action: func(bs *model.AppModel) {
			viewName := "覆盖代理配置"
			template.ShowTextViewPage(bs, viewName, config.Sync(), nil)
		},
	},
	{
		Name: "屏蔽后台访问",
		Action: func(bs *model.AppModel) {
			viewName := "禁用后台访问"
			writer := bytes.Buffer{}
			config.DisableBackendAccess(&writer)
			template.ShowTextViewPage(bs, viewName, writer, nil)
		},
	},
	{
		Name: "[red]开启后台访问[white]",
		Action: func(bs *model.AppModel) {
			viewName := "开启后台访问"
			writer := bytes.Buffer{}
			config.EnableBackendAccess(&writer)
			template.ShowTextViewPage(bs, viewName, writer, nil)
		},
	},
	{
		Name: "启用nginx监控配置(POC用)",
		Action: func(bs *model.AppModel) {
			viewName := "启用nginx监控配置(POC用)"
			writer := bytes.Buffer{}
			config.EnableNginxStatus(&writer)
			template.ShowTextViewPage(bs, viewName, writer, nil)
		},
	},
	{
		Name: "禁用nginx监控配置(POC用)",
		Action: func(bs *model.AppModel) {
			viewName := "禁用nginx监控配置(POC用)"
			writer := bytes.Buffer{}
			config.DisableNginxStatus(&writer)
			template.ShowTextViewPage(bs, viewName, writer, nil)
		},
	},
}
