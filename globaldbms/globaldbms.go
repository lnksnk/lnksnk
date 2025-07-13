package globaldbms

import (
	"database/sql"

	"github.com/lnksnk/lnksnk/dbms"
	"github.com/lnksnk/lnksnk/globalfs"
)

var DBMS dbms.DBMS
var DRIVERS dbms.Drivers
var CONNNECTIONS dbms.Connections

func MapDriver(name string, drvragrs ...interface{}) {
	if name != "" && len(drvragrs) > 0 {

		var dbinvke dbms.InvokeDBFunc
		var prssqlargs dbms.ParseSqlArgFunc
		for _, d := range drvragrs {
			if dbinvked, dbinvkek := d.(func(string, ...interface{}) (*sql.DB, error)); dbinvkek {
				if dbinvke == nil && dbinvked != nil {
					dbinvke = dbinvked
				}
				continue
			}
			if prssqlargsd, prssqlargsk := d.(func(int) string); prssqlargsk {
				if prssqlargs == nil && prssqlargsd != nil {
					prssqlargs = prssqlargsd
				}
				continue
			}
		}
		if dbinvke != nil {
			DRIVERS.Register(name, []interface{}{dbinvke, prssqlargs})
		}
	}
}

func init() {
	DBMS = dbms.NewDBMS(globalfs.FSYS)
	DRIVERS = DBMS.Drivers()
	CONNNECTIONS = DBMS.Connections()
}
