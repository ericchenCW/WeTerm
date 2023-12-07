package template

import (
	"bufio"
	"fmt"
	"net/http"
	"weterm/model"

	"github.com/rivo/tview"
)

func ShowElasticExecutePage(receiver *model.AppModel, title string, shellCommand string) {
	outputTextView := tview.NewTextView().SetDynamicColors(true)
	outputTextView.SetBorder(true).SetTitle(title).SetTitleAlign(tview.AlignCenter)
	outputTextView.SetScrollable(true)

	// 请求elasticsearch数据并获取输出
	esUser := "elastic"
	esPassword := "0r7C5NSoqyR9"
	esHost := "http://10.10.26.235:9200"

	client := &http.Client{}
	// request with basic auth
	req, err := http.NewRequest("GET", esHost, nil)
	if err != nil {
		fmt.Fprintf(outputTextView, "Error: %v\n", err)
		return
	}

	req.SetBasicAuth(esUser, esPassword)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(outputTextView, "Error: %v\n", err)
		return
	}

	//将输出写入stdout
	stdout := bufio.NewReader(resp.Body)

	// 创建一个页面
	receiver.CorePages.AddPage("elasticsearch_execute_page", outputTextView, true, false)
	receiver.CorePages.SwitchToPage("elasticsearch_execute_page")

	// start a goroutine to read the command's output
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			// use the application's queue to update the UI
			receiver.CoreApp.QueueUpdateDraw(func() {
				fmt.Fprintf(outputTextView, "Request Output: %s\n", scanner.Text())
				outputTextView.ScrollToEnd() // Scroll to end
			})
		}
		if err := scanner.Err(); err != nil {
			receiver.CoreApp.QueueUpdateDraw(func() {
				fmt.Fprintf(outputTextView, "Error: %v\n", err)
			})
		}
	}()
}
