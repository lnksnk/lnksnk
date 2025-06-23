package globalsqlserver

import (
	"github.com/lnksnk/lnksnk/dbms/mssql"
	"github.com/lnksnk/lnksnk/globaldbms"
)

func init() {
	globaldbms.MapDriver("mssql", mssql.InvokeDB, mssql.ParseSqlParam)
	globaldbms.MapDriver("sqlserver", mssql.InvokeDB, mssql.ParseSqlParam)
}
