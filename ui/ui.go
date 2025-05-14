package ui

import (
	"embed"

	"github.com/lnksnk/lnksnk/fs"

	"github.com/lnksnk/lnksnk/ioext"
)

//go:embed js/*.*
var UiJsFS embed.FS

func ImportUiJS(fsys fs.MultiFileSystem, altroot ...string) {
	fs.ImportResource(func(srcroot string, src *ioext.Buffer, srcfsys fs.MultiFileSystem) {
		srcfsys.Map(srcroot)
		if !srcfsys.Exist(srcroot + "/index.html") {
			srcfsys.Set(srcroot+"/index.html", src)
		} else if len(altroot) > 0 && altroot[0] != "" && altroot[0] != "/" && altroot[0][0] == '/' {
			if altroot[0][len(altroot[0])-1] == '/' {
				altroot[0] = altroot[0][:len(altroot[0])-1]
			}
			if altroot[0] != srcroot {
				srcfsys.Map(altroot[0])
				srcfsys.Set(altroot[0]+"/index.html", src)
			}
		}
	}, fsys, UiJsFS, ".js", ".go", true, "/ui", "js")
}
