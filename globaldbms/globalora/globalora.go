package globalora

import (
	"github.com/lnksnk/lnksnk/dbms/ora"
	"github.com/lnksnk/lnksnk/globaldbms"
)

func init() {
	globaldbms.MapDriver("oracle", ora.InvokeDB, ora.ParseSqlParam)
}
