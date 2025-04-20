package fonts

import (
	"embed"

	fs "github.com/lnksnk/lnksnk/fs"
	fsembed "github.com/lnksnk/lnksnk/fs/embed"
	"github.com/lnksnk/lnksnk/ioext"
)

type FSFonts struct {
	fsembed.EmbedFSOpen
	fsembed.EmbedFSReadDir
	fsembed.EmbedFSReadFile
}

//go:embed *
var FontsFS embed.FS

func EmbedFonts(fsys fs.MultiFileSystem) {
	fsembed.ImportResource(func(srcroot string, src *ioext.Buffer, srcfsys fs.MultiFileSystem) {
		srcfsys.Map(srcroot)
		srcfsys.Set(srcroot+"/index.html", src)
	}, fsys, FontsFS, ".css", ".go", true, "/fonts", "material", "roboto")
}
