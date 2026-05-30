package aio

import (
	_ "embed"
)

//go:embed assets/init_ip.sh
var InitIPScript string

//go:embed assets/stop.sh
var StopScript string

//go:embed assets/init_topo.sh
var InitTopoScript string
