package globalmysql

import (
	"github.com/lnksnk/lnksnk/dbms/mysql"
	"github.com/lnksnk/lnksnk/globaldbms"
)

func init() {
	globaldbms.MapDriver("mssql", mysql.InvokeDB, mysql.ParseSqlParam)
}
