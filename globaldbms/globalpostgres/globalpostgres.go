package globalpostgres

import (
	"github.com/lnksnk/lnksnk/dbms/postgres"
	"github.com/lnksnk/lnksnk/globaldbms"
)

func init() {
	globaldbms.MapDriver("postgres", postgres.InvokeDB, postgres.ParseSqlParam)

}
