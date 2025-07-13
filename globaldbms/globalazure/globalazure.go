package globalazure

import (
	"github.com/lnksnk/lnksnk/dbms/mssql"
	"github.com/lnksnk/lnksnk/globaldbms"
)

func init() {
	globaldbms.DRIVERS.Register("azuresql", mssql.InvokeDBAzure, mssql.ParseSqlParam)
}
