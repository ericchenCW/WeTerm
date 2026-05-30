package index

import (
	"weterm/model"
	inspectpage "weterm/pages/inspect"
)

// componentHealthMenu 是「服务概览」子菜单。原先基于 pages/healthcheck 的弱实现
// （仅 host/consul/mysql）已被 weops-inspect 取代：这里改为 in-process 调用
// inspect 的主机指标速查（RunHostsOnly），与「平台巡检」的全量巡检共用同一引擎，
// 区别在于概览只采主机、不探开源组件、不出 HTML 报告。
var componentHealthMenu = []MenuItem{
	{
		Name: "主机健康速查",
		Action: func(bs *model.AppModel) {
			inspectpage.ShowOverviewPage(bs)
		},
	},
}
