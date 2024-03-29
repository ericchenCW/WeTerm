package index

import (
	"time"
	"weterm/model"
	"weterm/pages/collect"
	"weterm/pages/template"
	"weterm/pages/template/table"
)

var collectMenu = []MenuItem{
	{
		Name: "Paas和Saas版本信息",
		Action: func(bs *model.AppModel) {
			viewName := "Paas和Saas版本信息"
			t := table.NewTable(viewName)
			tableData := collect.GetAppVersion()
			t.Update(&tableData)
			t.BuildTable(bs, viewName, time.Duration(1), nil, nil)
		},
	},
	{
		Name: "镜像版本信息",
		Action: func(bs *model.AppModel) {
			viewName := "镜像版本信息"
			t := table.NewTable(viewName)
			tableData := collect.GetImageVersion()
			t.Update(&tableData)
			t.BuildTable(bs, viewName, time.Duration(1), nil, nil)
		},
	},
	{
		Name: "日志文件采集",
		Action: func(bs *model.AppModel) {
			template.ShowShellExecutePage(bs, "日志文件采集", collect.SyncLogScript)
		},
	},
}
