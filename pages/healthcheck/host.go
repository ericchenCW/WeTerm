package healthcheck

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"weterm/pages/template/table"
	"weterm/utils"

	"github.com/rs/zerolog/log"
)

type HostHealth struct {
	BaseHealthChecker
	baseScript   string
	detailScript string
}

type ServerData struct {
	CPUPercent    string    `json:"-"`
	CPUPercentAUX float64   `json:"cpu_percent"`
	CPULoad       string    `json:"-"`
	CPULoadAUX    []float64 `json:"cpu_load"`
	Memory        Memory    `json:"memory"`
	Swap          Memory    `json:"swap"`
	Time          string    `json:"datetime"`
}

type Memory struct {
	Total      string  `json:"total"`
	Used       string  `json:"used"`
	Free       string  `json:"free"`
	Percent    string  `json:"-"`
	PercentAUX float64 `json:"percent"`
	Available  string  `json:"available"`
}

type Disk struct {
	Total   string  `json:"total"`
	Used    string  `json:"used"`
	Free    string  `json:"free"`
	Percent float64 `json:"percent"`
}

type HostDetail struct {
	IP               string            `json:"-"`
	Memory           string            `json:"memory_info"`
	Swap             string            `json:"swap_info"`
	LinuxDistro      string            `json:"linux_distro"`
	KernelVersion    string            `json:"kernel_version"`
	Hosts            []string          `json:"hosts"`
	NetworkInterface map[string]string `json:"network_interfaces"`
	DNSServers       []string          `json:"dns_servers"`
	CPUDetail        CPUDetail         `json:"cpu_info"`
	Disk             map[string]Disk   `json:"disk_usages"`
}

type CPUDetail struct {
	Brand string `json:"brand_model"`
	Cores string `json:"cores"`
}

//go:embed asserts/host_base.py
var baseScript string

//go:embed asserts/host_detail.py
var detailScript string

//go:embed asserts/host_detail.tpl
var detailTemplate string

func NewHostHealth() HostHealth {
	return HostHealth{
		baseScript:   baseScript,
		detailScript: detailScript,
	}
}

func (h HostHealth) Check() table.TableData {
	hosts := strings.Split(os.Getenv("ALL_IP_COMMA"), ",")
	result := table.TableData{Header: table.Header{
		table.HeaderColumn{Name: "IP"},
		table.HeaderColumn{Name: "%CPU"},
		table.HeaderColumn{Name: "%MEM"},
		table.HeaderColumn{Name: "%SWAP"},
		table.HeaderColumn{Name: "LOAD AVG"},
		table.HeaderColumn{Name: "TIME"},
	}}
	ch := make(chan table.Row, len(hosts))
	var wg sync.WaitGroup
	for host := range hosts {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			ch <- h.toBasicRow(utils.RunSSH(s, h.baseScript, "python"), s)
		}(hosts[host])
	}
	wg.Wait()
	close(ch)
	for row := range ch {
		result.Rows = append(result.Rows, row)
	}
	sort.Slice(result.Rows, func(i, j int) bool {
		return result.Rows[i].Fields[0] < result.Rows[j].Fields[0]
	})
	return result
}

func (h HostHealth) Detail(host string) bytes.Buffer {
	var detail HostDetail
	detailBytes := utils.RunSSH(host, h.detailScript, "python")
	err := json.Unmarshal(detailBytes, &detail)
	detail.IP = host
	if err != nil {
		log.Logger.Err(err).Msg("Unmarshal Host Detail failed")
	}
	t := template.New("detail template")
	t, err = t.Parse(detailTemplate)
	if err != nil {
		log.Logger.Err(err).Msg("Host template parse Detail failed")
	}

	var tpl bytes.Buffer
	err = t.Execute(&tpl, detail)
	if err != nil {
		log.Logger.Err(err).Msg("Host template render Detail failed")
	}
	return tpl
}

func (h HostHealth) toBasicRow(raw []byte, host string) table.Row {
	var msg ServerData
	err := json.Unmarshal(raw, &msg)
	if err != nil {
		log.Error().Err(err).Msg("Unmarshal host row failed")
	}
	log.Debug().
		Str("IP", host).
		Float64("CPUPercentAUX", msg.CPUPercentAUX).
		Float64("Memory.PrcentAUX", msg.Memory.PercentAUX).
		Str("CPULoad", msg.CPULoad).
		Msg("Build Host Health Row")

	row := table.NewRow(6)
	row.ID = host
	row.Fields[0] = host
	row.Fields[1] = msg.CPUPercent
	row.Fields[2] = msg.Memory.Percent
	row.Fields[3] = msg.Swap.Percent
	row.Fields[4] = msg.CPULoad
	row.Fields[5] = "[aqua]" + msg.Time
	return row
}

func (sd *ServerData) UnmarshalJSON(data []byte) error {
	type Alias ServerData
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(sd),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	sd.CPUPercent = sd.buildColorPercent(sd.CPUPercentAUX)
	sd.CPULoad = sd.float64SliceToString(sd.CPULoadAUX, "%.2f")
	sd.Memory.Percent = sd.buildColorPercent(sd.Memory.PercentAUX)
	sd.Swap.Percent = sd.buildColorPercent(sd.Swap.PercentAUX)
	return nil
}

func (sd *ServerData) buildColorPercent(f float64) string {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if f <= 50 {
		return "[green]" + s
	} else if 50 < f && f <= 90 {
		return "[orange]" + s
	} else {
		return "[red]" + s
	}
}

func (sd *ServerData) float64SliceToString(f []float64, format string) string {
	s := make([]string, len(f))
	for i, v := range f {
		s[i] = fmt.Sprintf(format, v)
	}
	return strings.Join(s, ", ")
}

func (h *HostHealth) SaveHostDetails() string {
	var details []HostDetail
	hosts := strings.Split(os.Getenv("ALL_IP_COMMA"), ",")
	ch := make(chan []byte, len(hosts))
	var wg sync.WaitGroup
	for host := range hosts {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			ch <- utils.RunSSH(s, h.baseScript, "python")
		}(hosts[host])
	}
	wg.Wait()
	close(ch)
	for raw := range ch {
		var detail HostDetail
		err := json.Unmarshal(raw, &detail)
		if err != nil {
			log.Logger.Err(err).Msg("Unmarshal Host Raw failed")
			continue
		}
		details = append(details, detail)
	}
	jsonData, err := json.MarshalIndent(details, "", " ")
	if err != nil {
		log.Logger.Err(err).Msg("Marshal Host Raw failed")
		return "Marshal Host Raw failed"
	}
	dir, err := os.MkdirTemp("/tmp", "weterm_*")
	if err != nil {
		log.Logger.Err(err).Msg("Make temp dir failed")
		return "Marshal Host Raw failed"
	}
	fileName := "hosts_detail.json"
	path := dir + fileName
	err = os.WriteFile(path, jsonData, 0644)
	if err != nil {
		log.Logger.Err(err).Str("Path", path).Msg("Save File failed")
		return "Save File failed"
	}
	return "保存成功，文件目录:" + path
}
