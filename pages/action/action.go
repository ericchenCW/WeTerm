package action

import (
	_ "embed"
)

//go:embed asserts/unseal_vault.sh
var UnsealVaultScript string

//go:embed asserts/reload_casbin.sh
var ReloadCasbin string
