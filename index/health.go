package index

import (
	"time"
	"weterm/model"
	"weterm/pages/healthcheck"
	"weterm/pages/template"
	"weterm/pages/template/table"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

var componentHealthMenu = []MenuItem{
	{
		Name: "主机",
		Action: func(bs *model.AppModel) {
			viewName := "主机性能概览-每3秒刷新"
			refreshPeriod := 3
			h := healthcheck.NewHostHealth()
			t := table.NewTable(viewName)
			check := table.RefreshFunction(h.Check)
			selectedFunc := func(row, col int) {
				if row == 0 {
					return
				}
				host := t.GetCell(row, 0).Text
				log.Logger.Debug().Int("row", row).Int("col", col).Str("host", host).Msg("In SelectedFunction")
				doneFunction := func(key tcell.Key) {
					log.Logger.Info().Msg("key down")
					// if key == tcell.KeyF10 {
					// 	filePath := h.SaveHostDetails()
					// 	modal := tview.NewModal().SetText(filePath).AddButtons([]string{"OK"}).SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					// 		if buttonLabel == "OK" {
					// 			bs.CorePages.RemovePage("modal")
					// 		}
					// 	})
					// 	bs.CorePages.AddPage("save_file_modal", modal, false, false)
					// 	bs.CorePages.SwitchToPage("save_file_modal")
					// }
				}
				template.ShowTextViewPage(bs, "主机基础信息-"+host, h.Detail(host), &doneFunction)
			}
			tableData := h.Check()
			t.Update(&tableData)
			t.BuildTable(bs, viewName, time.Duration(refreshPeriod), &check, &selectedFunc)
		},
	},
	{
		Name: "consul集群",
		Action: func(bs *model.AppModel) {
			h := healthcheck.NewConsulHealth()
			viewName := "Consul集群概览-每1秒刷新"
			refreshPeriod := 1
			t := table.NewTable(viewName)
			check := table.RefreshFunction(h.Check)
			t.BuildTable(bs, viewName, time.Duration(refreshPeriod), &check, nil)
		},
	},
	{
		Name: "consul服务",
		Action: func(bs *model.AppModel) {
			h := healthcheck.NewServiceHealth()
			viewName := "Consul服务概览-每10秒刷新"
			refreshPeriod := 10
			t := table.NewTable(viewName)
			tableData := h.Check()
			t.Update(&tableData)
			check := table.RefreshFunction(h.Check)
			selectedFunction := func(row, col int) {
				if row == 0 {
					return
				}
				serviceID := t.GetCell(row, 1).Text
				output := h.Detail(serviceID)
				modal := tview.NewModal().SetText("详情: " + output).AddButtons([]string{"OK"}).SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					if buttonLabel == "OK" {
						bs.CorePages.RemovePage("modal")
					}
				})
				bs.CorePages.AddPage("modal", modal, false, false)
				bs.CorePages.SwitchToPage("modal")
			}
			t.BuildTable(bs, viewName, time.Duration(refreshPeriod), &check, &selectedFunction)
		},
	},
	// {
	// 	Name: "mysql-未实现",
	// 	Action: func(bs *model.AppModel) {
	// 		m := healthcheck.NewMysqlHealth("mysql-default.service.consul", "root", os.Getenv("BK_MYSQL_ADMIN_PASSWORD"), "mysql")
	// 		template.ShowHealthView(bs, m)
	// 	},
	// },
	// {
	// 	Name: "redis-未实现",
	// 	Action: func(bs *model.AppModel) {
	// 		pages.SetUpStatusPage(bs)
	// 		bs.CorePages.SwitchToPage("status_check")
	// 	},
	// },
	// {
	// 	Name: "mongodb-未实现",
	// 	Action: func(bs *model.AppModel) {
	// 		pages.SetUpStatusPage(bs)
	// 		bs.CorePages.SwitchToPage("status_check")
	// 	},
	// },
	// {
	// 	Name: "...",
	// 	Action: func(bs *model.AppModel) {
	// 		pages.SetUpStatusPage(bs)
	// 		bs.CorePages.SwitchToPage("status_check")
	// 	},
	// },
}

// var serviceHealthMenu = []MenuItem{
// 	{
// 		Name: "服务概览",
// 		Action: func(bs *model.AppModel) {
// 			h := healthcheck.NewConsulHealth()
// 			viewName := "Consul概览-每10秒刷新"
// 			refreshPeriod := 10
// 			t := table.NewTable(viewName)
// 			t.BuildTable(bs, viewName, time.Duration(refreshPeriod), h.Check)
// 		},
// 	},
// 	{
// 		Name: "Paas-未实现",
// 		Action: func(bs *model.AppModel) {
// 			pages.SetUpStatusPage(bs)
// 			bs.CorePages.SwitchToPage("status_check")
// 		},
// 	},
// 	{
// 		Name: "用户管理-未实现",
// 		Action: func(bs *model.AppModel) {
// 			pages.SetUpStatusPage(bs)
// 			bs.CorePages.SwitchToPage("status_check")
// 		},
// 	},
// 	{
// 		Name: "权限中心-未实现",
// 		Action: func(bs *model.AppModel) {
// 			pages.SetUpStatusPage(bs)
// 			bs.CorePages.SwitchToPage("status_check")
// 		},
// 	},
// 	{
// 		Name: "CMDB-未实现",
// 		Action: func(bs *model.AppModel) {
// 			pages.SetUpStatusPage(bs)
// 			bs.CorePages.SwitchToPage("status_check")
// 		},
// 	},
// 	{
// 		Name: "作业平台-未实现",
// 		Action: func(bs *model.AppModel) {
// 			pages.SetUpStatusPage(bs)
// 			bs.CorePages.SwitchToPage("status_check")
// 		},
// 	},
// 	{
// 		Name: "监控平台-未实现",
// 		Action: func(bs *model.AppModel) {
// 			pages.SetUpStatusPage(bs)
// 			bs.CorePages.SwitchToPage("status_check")
// 		},
// 	},
// 	{
// 		Name: "WeOps组件-未实现",
// 		Action: func(bs *model.AppModel) {
// 			pages.SetUpStatusPage(bs)
// 			bs.CorePages.SwitchToPage("status_check")
// 		},
// 	},
// 	{
// 		Name: "...",
// 		Action: func(bs *model.AppModel) {
// 			pages.SetUpStatusPage(bs)
// 			bs.CorePages.SwitchToPage("status_check")
// 		},
// 	},
// }
