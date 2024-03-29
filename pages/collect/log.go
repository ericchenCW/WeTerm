package collect

import (
	_ "embed"
)

//go:embed asserts/sync_log.sh
var SyncLogScript string
