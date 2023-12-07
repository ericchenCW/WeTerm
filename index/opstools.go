package index

import (
	"weterm/model"
	"weterm/pages/template"
)

var opsMenu = []MenuItem{
	{
		Name: "查看主机进程",
		Action: func(bs *model.AppModel) {
			template.ShowShellExecutePage(bs, "进程查询", "curl -u elastic:***REMOVED*** http://***REDACTED-IP***:9200/_cat/nodes?v")
		},
	},
}
