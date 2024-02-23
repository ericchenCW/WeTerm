package utils

import (
	"database/sql"
	"fmt"
	"os"
	"sync"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog/log"
)

var (
	db   *sql.DB
	once sync.Once
	err  error
)

func GetDBIntance() (*sql.DB, error) { // 返回错误信息
	once.Do(func() {
		uName := "root"
		uPass := os.Getenv("BK_MYSQL_ADMIN_PASSWORD")
		host := "mysql-default.service.consul"
		dbName := "mysql"
		dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s", uName, uPass, host, dbName)
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			log.Logger.Err(err).Msg("Init MYSQL connection failed.")
			return
		}

		err = db.Ping()
		if err != nil {
			log.Logger.Err(err).Msg("Database accessible error.")
			db = nil
			return
		}
	})

	if db == nil || err != nil {
		return nil, err
	}

	return db, nil
}

func MysqlQuery[T any](q string, parse func(rows *sql.Rows) T) ([]T, error) {
	db, err := GetDBIntance()
	if err != nil {
		log.Logger.Err(err).Msg("Get DB inst failed")
	}

	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		results = append(results, parse(rows))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
