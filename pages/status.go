package pages

import (
	"fmt"
	"github.com/rivo/tview"
	"os/exec"
	"strconv"
	"strings"
	"weterm/model"
	"weterm/utils"
)

func SetUpStatusPage(receiver *model.AppModel) {
	cmd := exec.Command("bash", "-c", "ulimit -n")
	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	ulimit, _ := strconv.Atoi(strings.TrimSpace(string(output)))

	// 创建一个文本框用于显示ulimit数量和emoji表示
	outputTextView := tview.NewTextView().SetTextAlign(tview.AlignLeft).SetDynamicColors(true)

	if ulimit > 1000 {
		outputTextView.SetText(utils.MakeHealthText(fmt.Sprintf("ulimit: %d", ulimit)))
	} else {
		outputTextView.SetText(utils.MakeWarnText(fmt.Sprintf("ulimit: %d", ulimit)))
	}

	// 创建一个布局，并将文本框添加到其中
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(outputTextView, 0, 1, true)

	// 创建一个页面
	receiver.CorePages.AddPage("status_check", layout, true, false)
}
