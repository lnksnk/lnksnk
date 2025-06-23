package globaldlv

import (
	"github.com/lnksnk/lnksnk/dbms/dlv"
	"github.com/lnksnk/lnksnk/globaldbms"
	"github.com/lnksnk/lnksnk/globalfs"
)

func init() {
	globaldbms.MapDriver("csv", dlv.InvokeCSVDB, dlv.ParseSqlParam)
	globaldbms.MapDriver("dlv", dlv.InvokeDLVDB, dlv.ParseSqlParam)
	globaldbms.DRIVERS.Register("csv", globalfs.GLOBALFS)
}
