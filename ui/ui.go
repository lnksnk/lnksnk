package ui

import (
	"embed"

	"github.com/lnksnk/lnksnk/fs"
	fsembed "github.com/lnksnk/lnksnk/fs/embed"
	"github.com/lnksnk/lnksnk/iorw"
)

//go:embed js/*.*
var UiJsFS embed.FS

func EmbedUiJS(fsys fs.MultiFileSystem) {
	fsembed.ImportResource(func(srcroot string, src *iorw.Buffer, srcfsys fs.MultiFileSystem) {
		srcfsys.Map(srcroot)
		srcfsys.Set(srcroot+"/index.html", src)
	}, fsys, UiJsFS, ".js", ".go", true, "/ui", "js")
}
