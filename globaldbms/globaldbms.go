package globaldbms

import (
	"database/sql"

	"github.com/lnksnk/lnksnk/dbms"
	"github.com/lnksnk/lnksnk/globalfs"
)

var DBMS dbms.DBMS
var DRIVERS dbms.Drivers
var CONNNECTIONS dbms.Connections
var DBDrivers = map[string][]interface{}{}

func MapDriver(name string, drvragrs ...interface{}) {
	if name != "" && len(drvragrs) > 0 {
		if _, dvrnmeok := DBDrivers[name]; !dvrnmeok {
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
				DBDrivers[name] = []interface{}{dbinvke, prssqlargs}
			}
		}
	}

}

func init() {
	DBMS = dbms.NewDBMS(globalfs.GLOBALFS)
	DRIVERS = DBMS.Drivers()
	DRIVERS.DefaultInvokable(func(driver string) (InvokeDB dbms.InvokeDBFunc, ParseSqlParam dbms.ParseSqlArgFunc) {
		if dbdvrapi, dbdrvapik := DBDrivers[driver]; dbdrvapik {
			InvokeDB, _ = dbdvrapi[0].(dbms.InvokeDBFunc)
			ParseSqlParam, _ = dbdvrapi[1].(dbms.ParseSqlArgFunc)
		}
		return
	})
	CONNNECTIONS = DBMS.Connections()
}
