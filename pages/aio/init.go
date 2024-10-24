package aio

import (
	_ "embed"
)

//go:embed asserts/init_ip.sh
var InitIPScript string

//go:embed asserts/stop.sh
var StopScript string

//go:embed asserts/init_topo.sh
var InitTopoScript string
