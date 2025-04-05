package mssql

import (
	"database/sql"
	"fmt"
	"strings"

	//helper registration sql server driver

	_ "github.com/microsoft/go-mssqldb"
	"github.com/microsoft/go-mssqldb/azuread"
	"github.com/pkg/errors"
)

// Open -wrap sql.Open("sqlserver", datasource)
func Open(datasource string) (*sql.DB, error) {
	var tlsversion = ""
	for _, dtasrc := range strings.Split(datasource, ";") {
		if strings.HasPrefix(dtasrc, "tlsmin=") {
			if tlsversion = strings.TrimSpace(dtasrc[len("tlsmin="):]); tlsversion == "" {
				tlsversion = "1.0"
				datasource = strings.Replace(datasource, "tlsmin=", "tlsmin="+tlsversion, 1)
			}
		}
	}
	if tlsversion == "" {
		tlsversion = "1.0"
		datasource += ";" + "tlsmin=" + tlsversion
	}

	return sql.Open("sqlserver", datasource)
}

// Open -wrap sql.Open("azure", datasource)
func OpenAzure(datasource string) (*sql.DB, error) {
	var tlsversion = ""
	for _, dtasrc := range strings.Split(datasource, ";") {
		if strings.HasPrefix(dtasrc, "tlsmin=") {
			if tlsversion = strings.TrimSpace(dtasrc[len("tlsmin="):]); tlsversion == "" {
				tlsversion = "1.0"
				datasource = strings.Replace(datasource, "tlsmin=", "tlsmin="+tlsversion, 1)
			}
		}
	}
	if tlsversion == "" {
		tlsversion = "1.0"
		datasource += ";" + "tlsmin=" + tlsversion
	}
	return sql.Open(azuread.DriverName, datasource)
}

func parseSqlParam(totalArgs int) (s string) {
	return ("@p" + fmt.Sprintf("%d", totalArgs))
}

func ParseSqlParam(totalArgs int) (s string) {
	return "$" + fmt.Sprintf("%d", totalArgs+1)
}

func InvokeDB(datasource string, a ...interface{}) (db *sql.DB, err error) {
	db, err = Open(datasource)
	if err != nil {
		return nil, errors.Wrap(err, "create db conn pool")
	}
	return
}

func InvokeDBAzure(datasource string, a ...interface{}) (db *sql.DB, err error) {
	db, err = OpenAzure(datasource)
	if err != nil {
		return nil, errors.Wrap(err, "create db conn pool")
	}
	return
}
