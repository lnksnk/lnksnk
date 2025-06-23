package globalazure

import (
	"github.com/lnksnk/lnksnk/dbms/mssql"
	"github.com/lnksnk/lnksnk/globaldbms"
)

func init() {
	globaldbms.MapDriver("azuresql", mssql.InvokeDBAzure, mssql.ParseSqlParam)
}
