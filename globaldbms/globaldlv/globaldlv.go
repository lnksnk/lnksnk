package globaldlv

import (
	"github.com/lnksnk/lnksnk/dbms/dlv"
	"github.com/lnksnk/lnksnk/globaldbms"
)

func init() {
	globaldbms.MapDriver("csv", dlv.InvokeCSVDB, dlv.ParseSqlParam)
	globaldbms.MapDriver("dlv", dlv.InvokeDLVDB, dlv.ParseSqlParam)
}
