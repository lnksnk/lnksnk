package mysql

import (
	"database/sql"

	//helper registration mysql server driver
	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
)

// Open -wrap sql.Open("mysql", datasource)
func Open(datasource string) (*sql.DB, error) {
	return sql.Open("mysql", datasource)
}

func ParseSqlParam(totalArgs int) (s string) {
	return "?"
}

func InvokeDB(datasource string, a ...interface{}) (db *sql.DB, err error) {
	db, err = Open(datasource)
	if err != nil {
		return nil, errors.Wrap(err, "create db conn pool")
	}
	return
}
