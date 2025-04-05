package sqlite

import (
	"database/sql"
	"fmt"

	//helper registration sqlite driver

	"github.com/pkg/errors"
	_ "modernc.org/sqlite"
)

// Open -wrap sql.Open("sqlite", datasource)
// when registering driver "sqlite"
func Open(datasource string) (*sql.DB, error) {
	return sql.Open("sqlite", datasource)
}

func ParseSqlParam(totalArgs int) (s string) {
	return "$" + fmt.Sprintf("%d", totalArgs+1)
}

func InvokeDB(datasource string, a ...interface{}) (db *sql.DB, err error) {
	if datasource == ":memory:" {
		datasource = "file::memory:?mode=memory"
	}
	db, err = Open(datasource)
	if err != nil {
		return nil, errors.Wrap(err, "create db conn")
	}
	return
}
