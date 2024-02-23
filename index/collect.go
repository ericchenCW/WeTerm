package index

import (
	"time"
	"weterm/model"
	"weterm/pages/collect"
	"weterm/pages/template/table"
)

var collectMenu = []MenuItem{
	{
		Name: "版本信息",
		Action: func(bs *model.AppModel) {
			viewName := "版本信息"
			t := table.NewTable(viewName)
			tableData := collect.GetVersion()
			t.Update(&tableData)
			t.BuildTable(bs, viewName, time.Duration(1), nil, nil)
		},
	},
}
