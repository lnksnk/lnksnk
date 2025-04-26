package ui

import (
	"embed"

	"github.com/lnksnk/lnksnk/fs"

	"github.com/lnksnk/lnksnk/ioext"
)

//go:embed js/*.*
var UiJsFS embed.FS

func ImportUiJS(fsys fs.MultiFileSystem) {
	fs.ImportResource(func(srcroot string, src *ioext.Buffer, srcfsys fs.MultiFileSystem) {
		srcfsys.Map(srcroot)
		srcfsys.Set(srcroot+"/index.html", src)
	}, fsys, UiJsFS, ".js", ".go", true, "/ui", "js")
}
