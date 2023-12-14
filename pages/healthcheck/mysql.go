package healthcheck

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

type MysqlHealth struct {
	BaseHealthChecker
	db *sql.DB
}

func NewMysqlHealth(host string, uName string, uPass string, dbName string) MysqlHealth {
	// db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s", uName, uPass, host, dbName))

	// if err != nil {
	// 	log.Fatal(err)
	// }

	// if err = db.Ping(); err != nil {
	// 	log.Fatal(err)
	// }

	return MysqlHealth{}
}

func (m MysqlHealth) Check() []HealthResult {
	return []HealthResult{
		{status: Healthy, message: "Run query success"},
		{status: Warning, message: "Warning message"},
		{status: Error, message: "Error message"},
	}
}
