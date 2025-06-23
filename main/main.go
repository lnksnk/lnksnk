package main

import (
	"net/http"

	"github.com/lnksnk/lnksnk/fonts"
	_ "github.com/lnksnk/lnksnk/globaldbms/globalpostgres"
	_ "github.com/lnksnk/lnksnk/globaldbms/globalsqlite"
	"github.com/lnksnk/lnksnk/globalfs"
	"github.com/lnksnk/lnksnk/globalsession"
	"github.com/lnksnk/lnksnk/ui"

	"github.com/lnksnk/lnksnk/listen"
)

func main() {
	chn := make(chan bool, 1)
	var mltyfsys = globalfs.GLOBALFS
	mltyfsys.Map("/monaco", "C:/Users/evert/Downloads/monaco-editor-0.52.2.tgz")
	mltyfsys.Map("/threejs", "C:/Users/evert/Downloads/three.js-master.zip", true)
	mltyfsys.Map("/etl", "C:/projects/cim", true)
	mltyfsys.Map("/media", "C:/movies", true)
	fonts.ImportFonts(mltyfsys)
	ui.ImportUiJS(mltyfsys)
	var lstn = listen.NewListen(http.HandlerFunc(globalsession.HTTPSessionHandler))
	lstn.Serve("tcp", ":1089")
	<-chn
}
