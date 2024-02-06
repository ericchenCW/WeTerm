package healthcheck

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"weterm/pages/template/table"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

var tableHeader = table.Header{
	table.HeaderColumn{Name: "IP"},
	table.HeaderColumn{Name: "%CPU"},
	table.HeaderColumn{Name: "%MEM"},
	table.HeaderColumn{Name: "LOAD AVG"},
}

type HostHealth struct {
	BaseHealthChecker
	script string
}

type ServerData struct {
	CPUPercent     string          `json:"-"`
	CPUPercentAUX  float64         `json:"cpu_percent"`
	CPULoad        string          `json:"-"`
	CPULoadAUX     []float64       `json:"cpu_load"`
	Memory         Memory          `json:"memory"`
	Swap           Memory          `json:"swap"`
	Disk           map[string]Disk `json:"disk"`
	DiskIODelta    DiskIODelta     `json:"disk_io_delta"`
	NetworkIODelta NetworkIODelta  `json:"network_io_delta"`
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

type DiskIODelta struct {
	IOPS       int    `json:"IOPS"`
	Throughput string `json:"throughput"`
}

type NetworkIODelta struct {
	Sent     string `json:"sent"`
	Received string `json:"received"`
}

//go:embed asserts/host_base.py
var src string

func NewHostHealth() HostHealth {
	return HostHealth{
		script: src,
	}
}

func (h HostHealth) Check() table.TableData {
	hosts := strings.Split(os.Getenv("ALL_IP_COMMA"), ",")
	result := table.TableData{Header: tableHeader}
	ch := make(chan table.Row, len(hosts))
	var wg sync.WaitGroup
	for host := range hosts {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			ch <- h.runSSH(s)
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

func (h HostHealth) runSSH(host string) table.Row {
	privateKeyPath := os.Getenv("HOME") + "/.ssh/id_rsa"
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		log.Error().Err(err).Msg("ReadPrivateKey failed")
	}
	privateKey, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		log.Error().Err(err).Msg("ParsePrivateKey failed")
	}
	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(privateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", host), config)
	if err != nil {
		log.Error().Err(err).Msg("SSH Dial failed")
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		log.Error().Err(err).Msg("SSH Session failed")
	}
	defer session.Close()

	output, err := session.Output("python -c '" + h.script + "'")
	if err != nil {
		log.Error().Err(err).Msg("Run Script failed")
	}

	var msg ServerData
	err = json.Unmarshal(output, &msg)
	if err != nil {
		log.Error().Err(err).Msg("Unmarshal JSON failed")
	}
	log.Debug().
		Str("IP", host).
		Float64("CPUPercentAUX", msg.CPUPercentAUX).
		Float64("Memory.PrcentAUX", msg.Memory.PercentAUX).
		Str("CPULoad", msg.CPULoad).
		Msg("Build Host Health Row")

	row := table.NewRow(4)
	row.ID = host
	row.Fields[0] = "[aqua]" + host
	row.Fields[1] = msg.CPUPercent
	row.Fields[2] = msg.Memory.Percent
	row.Fields[3] = msg.CPULoad
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

// func (s ServerData) BuildMessage(ip string) string {
// 	r1_ip := fmt.Sprintf("[white]%s", ip)
// 	load_arr := strings.Join(strings.Fields(fmt.Sprint(s.CPULoad)), "")
// 	r2_cpu := fmt.Sprintf("        [aqua]CPU %%: %f        [aqua]CPU LOAD: %s        ", s.CPUPercent, load_arr)
// 	r3_mem := fmt.Sprintf(
// 		"        [aqua]MEM %%: [yellow]%f        [aqua]Total: [yellow]%s        [aqua]Used: [yellow]%s        [aqua]Free: [yellow]%s",
// 		s.Memory.Percent,
// 		s.Memory.Total,
// 		s.Memory.Used,
// 		s.Memory.Free,
// 	)
// 	r4_swap := fmt.Sprintf(
// 		"        [aqua]SWAP %%: [yellow]%f        [aqua]Total: [yellow]%s        [aqua]Used: [yellow]%s        [aqua]Free: [yellow]%s",
// 		s.Swap.Percent,
// 		s.Swap.Total,
// 		s.Swap.Used,
// 		s.Swap.Free,
// 	)
// 	r5_diskio := fmt.Sprintf(
// 		"        [aqua]IOPS: [yellow]%d        [aqua]Throughput: [yellow]%s",
// 		s.DiskIODelta.IOPS,
// 		s.DiskIODelta.Throughput,
// 	)
// 	r6_netio := fmt.Sprintf(
// 		"        [aqua]Sent: [yellow]%s        [aqua]Received: [yellow]%s",
// 		s.NetworkIODelta.Sent,
// 		s.NetworkIODelta.Received,
// 	)
// 	r7_disk := "        [White]Disk\n"
// 	for device := range s.Disk {
// 		str := fmt.Sprintf(
// 			"        [aqua]%s        [aqua]%%: [yellow]%f        [aqua]Total: [yellow]%s        [aqua]Used: [yellow]%s        [aqua]Free: [yellow]%s\n",
// 			device,
// 			s.Disk[device].Percent,
// 			s.Disk[device].Total,
// 			s.Disk[device].Used,
// 			s.Disk[device].Free,
// 		)
// 		r7_disk = r7_disk + str
// 	}
// 	body := fmt.Sprintf(
// 		"%s\n%s\n%s\n%s\n%s\n%s\n%s\n",
// 		r1_ip,
// 		r2_cpu,
// 		r3_mem,
// 		r4_swap,
// 		r5_diskio,
// 		r6_netio,
// 		r7_disk,
// 	)
// 	return body
// }
