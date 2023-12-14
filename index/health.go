package index

import (
	"weterm/model"
	"weterm/pages"
)

var componentHealthMenu = []MenuItem{
	{
		Name: "consul",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
	{
		Name: "mysql",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
	{
		Name: "redis",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
	{
		Name: "mongodb",
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
		Name: "Paas",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
	{
		Name: "用户管理",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
	{
		Name: "权限中心",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
	{
		Name: "CMDB",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
	{
		Name: "作业平台",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
	{
		Name: "监控平台",
		Action: func(bs *model.AppModel) {
			pages.SetUpStatusPage(bs)
			bs.CorePages.SwitchToPage("status_check")
		},
	},
	{
		Name: "WeOps组件",
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
