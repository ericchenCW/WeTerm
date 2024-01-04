package healthcheck

import (
	"database/sql"
	"fmt"

	"github.com/rs/zerolog/log"

	_ "github.com/go-sql-driver/mysql"
)

type MysqlHealth struct {
	BaseHealthChecker
	host string
	user string
	db   *sql.DB
}

func NewMysqlHealth(host string, uName string, uPass string, dbName string) MysqlHealth {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s", uName, uPass, host, dbName)
	db, err := sql.Open("mysql", dsn)

	if err != nil {
		log.Logger.Error().Err(err)
	}

	return MysqlHealth{db: db, host: host, user: uName}
}

func (m MysqlHealth) Check() []HealthResult {
	result := []HealthResult{}
	err := m.db.Ping()
	if err != nil {
		result = append(result, HealthResult{status: Error, message: err.Error()})
	} else {
		result = append(result, HealthResult{status: Healthy, message: fmt.Sprintf("MYSQL实例: %s 连接正常 用户名: %s", m.host, m.user)})
	}
	return result
}
