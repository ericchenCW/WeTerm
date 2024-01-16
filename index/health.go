package index

import (
	"os"
	"weterm/model"
	"weterm/pages"
	"weterm/pages/healthcheck"
	"weterm/pages/template"
)

var componentHealthMenu = []MenuItem{
	{
		Name: "consul",
		Action: func(bs *model.AppModel) {
			c := healthcheck.NewConsulHealth()
			template.ShowHealthView(bs, c)
		},
	},
	{
		Name: "mysql-未实现",
		Action: func(bs *model.AppModel) {
			m := healthcheck.NewMysqlHealth("mysql-default.service.consul", "root", os.Getenv("BK_MYSQL_ADMIN_PASSWORD"), "mysql")
			template.ShowHealthView(bs, m)
		},
	},
	{
		Name: "redis-未实现",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
	{
		Name: "mongodb-未实现",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
	{
		Name: "...",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
}

var serviceHealthMenu = []MenuItem{
	{
		Name: "服务概览",
		Action: func(bs *model.AppModel) {
			c := healthcheck.NewConsulHealth()
			template.ShowHealthView(bs, c)
		},
	},
	{
		Name: "Paas-未实现",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
	{
		Name: "用户管理-未实现",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
	{
		Name: "权限中心-未实现",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
	{
		Name: "CMDB-未实现",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
	{
		Name: "作业平台-未实现",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
	{
		Name: "监控平台-未实现",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
	{
		Name: "WeOps组件-未实现",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
	{
		Name: "...",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
}
