package index

import (
	"weterm/model"
	"weterm/pages"
	"weterm/pages/example"
	"weterm/pages/template"
)

var componentsOpsMenu = []MenuItem{
	{
		Name: "附加到容器",
		Action: func(bs *model.AppModel) {
			template.ShowShellExecutePage(bs, "进程查询", "curl -u elastic:ioHEmmAKOXdy http://10.10.26.235:9200/_cat/nodes?v")
		},
	},
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
			example.SetUpFormSamplePage(bs)
			bs.CorePages.SwitchToPage("form_sample")
		},
	},
}

var servicesOpsMenu = []MenuItem{
	{
		Name: "全部停止",
		Action: func(bs *model.AppModel) {
			template.ShowShellExecutePage(bs, "进程查询", "curl -u elastic:ioHEmmAKOXdy http://10.10.26.235:9200/_cat/nodes?v")
		},
	},
	{
		Name: "PaaS",
		Action: func(bs *model.AppModel) {
			template.ShowShellExecutePage(bs, "进程查询", "curl -u elastic:ioHEmmAKOXdy http://10.10.26.235:9200/_cat/nodes?v")
		},
	},
	{
		Name: "USERMGR",
		Action: func(bs *model.AppModel) {
			template.ShowShellExecutePage(bs, "进程查询", "curl -u elastic:ioHEmmAKOXdy http://10.10.26.235:9200/_cat/nodes?v")
		},
	},
	{
		Name: "IAM",
		Action: func(bs *model.AppModel) {
			template.ShowShellExecutePage(bs, "进程查询", "curl -u elastic:ioHEmmAKOXdy http://10.10.26.235:9200/_cat/nodes?v")
		},
	},
	{
		Name: "CMDB",
		Action: func(bs *model.AppModel) {
			template.ShowShellExecutePage(bs, "进程查询", "curl -u elastic:ioHEmmAKOXdy http://10.10.26.235:9200/_cat/nodes?v")
		},
	},
	{
		Name: "作业平台",
		Action: func(bs *model.AppModel) {
			template.ShowShellExecutePage(bs, "进程查询", "curl -u elastic:ioHEmmAKOXdy http://10.10.26.235:9200/_cat/nodes?v")
		},
	},
	{
		Name: "监控平台",
		Action: func(bs *model.AppModel) {
			template.ShowShellExecutePage(bs, "进程查询", "curl -u elastic:ioHEmmAKOXdy http://10.10.26.235:9200/_cat/nodes?v")
		},
	},
	{
		Name: "...",
		Action: func(bs *model.AppModel) {
			example.SetUpFormSamplePage(bs)
			bs.CorePages.SwitchToPage("form_sample")
		},
	},
}
