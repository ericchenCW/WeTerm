package aio

import (
	_ "embed"
)

//go:embed asserts/init_ip.sh
var InitIPScript string
