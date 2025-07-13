package globalfs

import (
	"github.com/lnksnk/lnksnk/es"
	"github.com/lnksnk/lnksnk/fonts"
	"github.com/lnksnk/lnksnk/fs"
	"github.com/lnksnk/lnksnk/fs/active"
	"github.com/lnksnk/lnksnk/ui"
)

func CompileProgram(fsys fs.MultiFileSystem, cde ...interface{}) (prgm interface{}, err error) {
	return es.CompileProgram(fsys, func(refscriptormod interface{}, modspecifier string) (rlsvdmodrec interface{}, rslvderr error) {
		if chdapi, _ := fsys.(interface {
			CachedInfo(path string) (chdfi active.CachedInfo, err error)
		}); chdapi != nil {
			chfi, chdfierr := chdapi.CachedInfo(modspecifier)
			if chdfierr != nil {
				rslvderr = chdfierr
				return
			}
			rlsvdmodrec = chfi.Program()
		}
		return
	}, cde...)
}

var FSYS fs.MultiFileSystem

func init() {
	FSYS = active.AciveFileSystem(CompileProgram)
	FSYS.CacheExtensions(".html", ".js", ".css", ".svg", ".woff2", ".woff", ".ttf", ".eot", ".sql")
	FSYS.DefaultExtensions(".html", ".js", ".json", ".css")
	FSYS.ActiveExtensions(".html", ".js", ".svg", ".json", ".xml", ".sql")
	fonts.ImportFonts(FSYS)
	ui.ImportUiJS(FSYS)
}
