package table_test

import (
	"encoding/json"
	"testing"
	"weterm/pages/healthcheck"
)

const (
	JSON = `
	{
		"cpu_percent": 9.9,
		"cpu_load": [
			2.62,
			2.31,
			2.27
		],
		"memory": {
			"total": "31.3GB",
			"available": "8.3GB",
			"percent": 73.3,
			"used": "22.0GB",
			"free": "1.1GB"
		},
		"swap": {
			"total": "7.9GB",
			"used": "3.4GB",
			"free": "4.5GB",
			"percent": 43.1
		},
		"disk": {
			"/dev/mapper/centos-root": {
				"total": "45.1GB",
				"used": "18.8GB",
				"free": "26.2GB",
				"percent": 41.8
			},
			"/dev/sda1": {
				"total": "1014.0MB",
				"used": "237.9MB",
				"free": "776.1MB",
				"percent": 23.5
			},
			"/dev/mapper/vg_data-lv_data": {
				"total": "196.7GB",
				"used": "130.7GB",
				"free": "56.0GB",
				"percent": 70.0
			}
		},
		"disk_io_delta": {
			"IOPS": 4,
			"throughput": "38.0KB"
		},
		"network_io_delta": {
			"sent": "649.9KB",
			"received": "921.7KB"
		}
	}
`
)

func TestUnmarshalJson(t *testing.T) {
	var msg healthcheck.ServerData
	if err := json.Unmarshal([]byte(JSON), &msg); err != nil {
		t.Error(err)
	}
}

func TestTableNew(t *testing.T) {
}
