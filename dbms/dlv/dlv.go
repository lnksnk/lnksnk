package dlv

import (
	"database/sql"

	"github.com/lnksnk/lnksnk/dbdvl"
	"github.com/pkg/errors"
)

// Open -wrap sql.Open("csv", datasource)
func OpenCSV(datasource string) (*sql.DB, error) {
	return sql.Open("csv", datasource)
}

// Open -wrap sql.Open("dlv", datasource)
func OpenDLV(datasource string) (*sql.DB, error) {
	return sql.Open("dlv", datasource)
}

func ParseSqlParam(totalArgs int) (s string) {
	return "?"
}

func InvokeCSVDB(datasource string, a ...interface{}) (db *sql.DB, err error) {
	db, err = OpenCSV(datasource)
	if err != nil {
		return nil, errors.Wrap(err, "create db conn pool")
	}
	return
}

func InvokeDLVDB(datasource string, a ...interface{}) (db *sql.DB, err error) {
	db, err = OpenDLV(datasource)
	if err != nil {
		return nil, errors.Wrap(err, "create db conn pool")
	}
	return
}

func init() {
	dbdvl.RegisterDriver("csv")
	dbdvl.RegisterDriver("dlv")
}
