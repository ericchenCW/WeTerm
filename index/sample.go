package index

import (
	"weterm/model"
	"weterm/pages/example"
	"weterm/pages/template"
)

var sampleMenu = []MenuItem{
	{
		Name: "基础表单",
		Action: func(bs *model.AppModel) {
			example.SetUpFormSamplePage(bs)
			bs.CorePages.SwitchToPage("form_sample")
		},
	},
	{
		Name: "表单Shell示例",
		Action: func(bs *model.AppModel) {
			formItems := []template.FormItem{
				{
					Type:        "input_field",
					Label:       "名称:",
					DefaultText: "Hello, world!",
					Validation: &template.Validation{
						Required: false,
						Type:     "none",
					},
				},
				{
					Type:        "input_field",
					Label:       "数值:",
					DefaultText: "",
					Validation: &template.Validation{
						Required: true,
						Type:     "numeric",
					},
				},
			}
			shellCommand := "echo {名称:} && echo {数值:}"
			template.ShowShellFormExecutePage(bs, "表单Shell示例", shellCommand, formItems)
			bs.CorePages.SwitchToPage("shell_form_execute")
		},
	},
	{
		Name: "Shell示例",
		Action: func(bs *model.AppModel) {
			example.SetUpShellCommandPage(bs)
			bs.CorePages.SwitchToPage("shell_command_page")
		},
	},
	{
		Name: "查看主机进程",
		Action: func(bs *model.AppModel) {
			template.ShowShellExecutePage(bs, "进程查询", "ps -ef")
		},
	},
	{
		Name: "查看日志",
		Action: func(bs *model.AppModel) {
			example.SetUpLogViewerPage(bs)
		},
	},
	{
		Name: "文本编辑",
		Action: func(bs *model.AppModel) {
			example.ShowEditFilePage(bs)
		},
	},
}
