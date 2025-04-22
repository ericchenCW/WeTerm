package action

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"weterm/model"

	"github.com/rs/zerolog/log"

	"github.com/google/uuid"
	"github.com/rivo/tview"
)

func genAlarmJson() string {
	currentTime := time.Now().UTC().Format(time.RFC3339)
	eventID := uuid.New().String()

	data := map[string]interface{}{
		"alerts": []map[string]interface{}{
			{
				"status": "resolved",
				"labels": map[string]string{
					"alertname":     "测试告警",
					"level":         "warning",
					"bk_biz_id":     "2",
					"field_cn_name": "测试指标",
					"event_id":      eventID,
					"strategy_name": "测试策略",
				},
				"annotations": map[string]string{
					"value":         "0.1",
					"alarm_content": "测试告警-weterm",
				},
				"startsAt": currentTime,
			},
		},
	}

	// 将数据编码为 JSON 字符串
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println("Error generating JSON:", err)
		return ""
	}
	return string(jsonData)
}

func SendAlarmView(receiver *model.AppModel) {
	if receiver.CorePages.HasPage("send_alarm_page") {
		receiver.CorePages.SwitchToPage("send_alarm_page")
	} else {
		output := tview.NewTextView().
			SetDynamicColors(true).
			SetRegions(true).
			SetWordWrap(true).
			ScrollToEnd()
		output.SetBorder(true).SetTitle("Output").SetTitleAlign(tview.AlignLeft)
		form := tview.NewForm()
		form.AddInputField("Attributes", "bk.biz.id=2,probe.name=0000-0000-0000-0000", 0, nil, nil)
		form.AddButton("Send Trace", func() {
			endpoint := form.GetFormItemByLabel("Endpoint").(*tview.InputField).GetText()
			attributes := form.GetFormItemByLabel("Attributes").(*tview.InputField).GetText()
			Send(endpoint, attributes, output)
		})
		layout := tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(form, 0, 1, true).
			AddItem(output, 0, 2, true)
		layout.SetBorder(true).SetTitle("发送告警").SetTitleAlign(tview.AlignCenter)
		receiver.CorePages.AddPage("send_alarm_page", layout, true, false)
		receiver.CorePages.SwitchToPage("send_alarm_page")
	}
}

func sendAlarm(alarm string) (bool, string) {
	URL := "http://paas.service.consul/o/cw_uac_saas/alarm/collect/event/common/13cce59d-d558-4612-8131-332d72c4fed9/"
	resp, err := http.Post(URL, "application/json", strings.NewReader(alarm))
	if err != nil {
		log.Logger.Error().Err(err).Msg("SendAlarm")
		return false, ""
	}
	if resp.StatusCode != http.StatusOK {
		log.Logger.Error().Msg("SendAlarm failed")
		return false, ""
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Logger.Err(err).Msg("ReadNginxStatus")
		return false, ""
	}
	return true, string(body)
}
