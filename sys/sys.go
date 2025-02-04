package main

import (
	"os"

	//_ "github.com/lnksnk/lnksnk/database/dbserve/connections"
	//_ "github.com/lnksnk/lnksnk/database/dbserve/drivers"
	//_ "github.com/lnksnk/lnksnk/database/dbserve/exec"
	//_ "github.com/lnksnk/lnksnk/database/dbserve/query"
	//_ "github.com/lnksnk/lnksnk/database/dbserve/register"
	//_ "github.com/lnksnk/lnksnk/database/dbserve/status"
	_ "github.com/lnksnk/lnksnk/database/mssql"
	_ "github.com/lnksnk/lnksnk/database/mysql"
	_ "github.com/lnksnk/lnksnk/database/ora"
	_ "github.com/lnksnk/lnksnk/database/postgres"
	_ "github.com/lnksnk/lnksnk/database/sqlite"

	//_ "github.com/lnksnk/lnksnk/emailservice/emailserve/imapcmd"
	_ "github.com/lnksnk/lnksnk/fonts"
	//"github.com/lnksnk/lnksnk/sys/app"
	"github.com/lnksnk/lnksnk/sys/srv"
	//"github.com/lnksnk/lnksnk/sys/webapp"
	_ "github.com/lnksnk/lnksnk/ui"
)

func main() {

	args := os.Args
	var appfunc func(...string) = nil

	appfunc = srv.Serve
	appfunc(args...)
}
