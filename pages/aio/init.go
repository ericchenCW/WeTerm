package aio

import (
	_ "embed"
)

//go:embed asserts/init_ip.sh
var InitIPScript string

//go:embed asserts/stop.sh
var StopScript string
