package action

import (
	_ "embed"
)

// ActionInfo 定义动作的信息
type ActionInfo struct {
	Name        string // 动作名称
	Description string // 动作描述
	Script      string // 脚本内容
}

//go:embed asserts/unseal_vault.sh
var unsealVaultScript string

//go:embed asserts/reload_casbin.sh
var reloadCasbinScript string

//go:embed asserts/backup_mongodb.sh
var backupMongodbScript string

//go:embed asserts/backup_mysql.sh
var backupMysqlScript string

//go:embed asserts/purge_rabbitmq_queues.sh
var purgeQueueScript string

// Actions 注册所有可用的动作
var Actions = map[string]ActionInfo{
	"reload_casbin": {
		Name:        "reload_casbin",
		Description: "Reload casbin mesh rules from WeOps",
		Script:      reloadCasbinScript,
	},
	"unseal_vault": {
		Name:        "unseal_vault",
		Description: "Unseal vault service",
		Script:      unsealVaultScript,
	},
	"backup_mongodb": {
		Name:        "backup_mongodb",
		Description: "Backup MongoDB database",
		Script:      backupMongodbScript,
	},
	"backup_mysql": {
		Name:        "backup_mysql",
		Description: "Backup MySQL database",
		Script:      backupMysqlScript,
	},
	"purge_rabbitmq": {
		Name:        "purge_rabbitmq",
		Description: "Purge RabbitMQ queues",
		Script:      purgeQueueScript,
	},
}

// GetAction 获取指定名称的动作信息
func GetAction(name string) (ActionInfo, bool) {
	action, exists := Actions[name]
	return action, exists
}

// GetAllActions 获取所有动作信息
func GetAllActions() map[string]ActionInfo {
	return Actions
}
