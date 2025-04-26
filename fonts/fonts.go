package fonts

import (
	"embed"

	fs "github.com/lnksnk/lnksnk/fs"
	"github.com/lnksnk/lnksnk/ioext"
)

//go:embed *
var FontsFS embed.FS

func ImportFonts(fsys fs.MultiFileSystem) {
	fs.ImportResource(func(srcroot string, src *ioext.Buffer, srcfsys fs.MultiFileSystem) {
		srcfsys.Map(srcroot)
		srcfsys.Set(srcroot+"/index.html", src)
	}, fsys, FontsFS, ".css", ".go", true, "/fonts", "material", "roboto")
}
