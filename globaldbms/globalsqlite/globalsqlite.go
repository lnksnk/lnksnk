package globalsqlite

import (
	"github.com/lnksnk/lnksnk/dbms/sqlite"
	"github.com/lnksnk/lnksnk/globaldbms"
)

func init() {
	globaldbms.MapDriver("sqlite", sqlite.InvokeDB, sqlite.ParseSqlParam)
}
