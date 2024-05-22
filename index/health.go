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
				serviceName := t.GetCell(row, 0).Text
				output := h.Detail(serviceID)
				modal := tview.NewModal().SetText("详情: " + output).AddButtons([]string{"重启", "取消"}).SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					if buttonLabel == "重启" {
						h.Restart(serviceName)
						bs.CorePages.RemovePage("modal")
					} else if buttonLabel == "取消" {
						bs.CorePages.RemovePage("modal")
					}
				})
				bs.CorePages.AddPage("modal", modal, false, false)
				bs.CorePages.SwitchToPage("modal")
			}
			t.BuildTable(bs, viewName, time.Duration(refreshPeriod), &check, &selectedFunction)
		},
	},
}
