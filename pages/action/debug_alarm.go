package action

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
	"weterm/model"

	"github.com/rivo/tview"
	logzero "github.com/rs/zerolog/log"

	"log"
)

func DebugAlarmView(receiver *model.AppModel) {
	if receiver.CorePages.HasPage("debug_alarm_page") {
		receiver.CorePages.SwitchToPage("debug_alarm_page")
	} else {
		output := tview.NewTextView().
			SetDynamicColors(true).
			SetRegions(true).
			SetWordWrap(true).
			ScrollToEnd()
		output.SetBorder(true).SetTitle("Output").SetTitleAlign(tview.AlignLeft)

		// 创建一个WaitGroup
		var wg sync.WaitGroup
		// 启动 DebugAlarm
		wg.Add(1)
		go DebugAlarm(output, &wg)

		// 启动定时器来滚动到末尾
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for range ticker.C {
				receiver.CoreApp.QueueUpdateDraw(func() {
					output.ScrollToEnd()
				})
			}
		}()

		// 等待 DebugAlarm 协程完成
		wg.Wait()
		// 添加页面并切换
		receiver.CorePages.AddPage("debug_alarm_page", output, true, false)
		receiver.CorePages.SwitchToPage("debug_alarm_page")
	}
}

func DebugAlarm(output *tview.TextView, wg *sync.WaitGroup) {
	defer wg.Done() // 确保 Done 在协程退出时被调用

	// 设置自定义日志记录器
	log.SetOutput(&TextViewLogger{view: output})
	log.Default().Println("Starting server...")

	// 启动http server 监听本地的9818端口
	http.HandleFunc("/notification", func(w http.ResponseWriter, r *http.Request) {
		// 输出源ip

		// 读取请求体
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Default().Println("Error reading request body:", err)
			return
		}
		defer r.Body.Close()

		// 美化接收到的 JSON 请求体内容
		var prettyBody interface{}
		if err := json.Unmarshal(body, &prettyBody); err != nil {
			log.Default().Println("Error unmarshalling request body:", err)
			return
		}

		// 格式化并打印请求体
		formattedBody, err := json.MarshalIndent(prettyBody, "", "  ")
		if err != nil {
			log.Default().Println("Error formatting JSON:", err)
			return
		}
		log.Default().Printf("Received request from [yellow]%s[white] ,Json Body: \n[green]%s[white] \n", r.RemoteAddr, formattedBody)

		// 返回响应
		w.WriteHeader(http.StatusOK)
		responseBody := `{
	"result": true,
	"data": "",
	"message": ""
}`
		fmt.Fprint(w, responseBody)
	})

	// 启动HTTP服务器
	go func() {
		if err := http.ListenAndServe(":9818", nil); err != nil {
			log.Default().Println("Failed to start server:", err)
			logzero.Error().Msg("failed to start server")
		}
	}()
	log.Default().Println("Server listening on http://127.0.0.1:9818/notification")
}
