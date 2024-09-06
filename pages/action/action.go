package action

import (
	_ "embed"
)

//go:embed asserts/unseal_vault.sh
var UnsealVaultScript string

//go:embed asserts/reload_casbin.sh
var ReloadCasbin string

//go:embed asserts/backup_mongodb.sh
var BackupMongodb string

//go:embed asserts/backup_mysql.sh
var BackupMysql string
