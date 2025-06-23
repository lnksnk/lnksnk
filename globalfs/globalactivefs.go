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

var GLOBALFS fs.MultiFileSystem

func init() {
	GLOBALFS = active.AciveFileSystem(CompileProgram)
	GLOBALFS.CacheExtensions(".html", ".js", ".css", ".svg", ".woff2", ".woff", ".ttf", ".eot", ".sql")
	GLOBALFS.DefaultExtensions(".html", ".js", ".json", ".css")
	GLOBALFS.ActiveExtensions(".html", ".js", ".svg", ".json", ".xml", ".sql")
	fonts.ImportFonts(GLOBALFS)
	ui.ImportUiJS(GLOBALFS)
}
