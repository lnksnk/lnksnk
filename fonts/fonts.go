package fonts

import (
	"embed"
	"io/fs"

	fsembed "github.com/lnksnk/lnksnk/fs/embed"
)

type FSFonts struct {
	fsembed.EmbedFSOpen
	fsembed.EmbedFSReadDir
	fsembed.EmbedFSReadFile
}

//go:embed *
var FontsFS embed.FS

func Fonts(paths ...string) fsembed.EmbedFS {
	if paths == nil {
		paths = append(paths, "roboto", "material")
	}
	return &struct {
		fsembed.EmbedFSOpen
		fsembed.EmbedFSReadDir
		fsembed.EmbedFSReadFile
	}{EmbedFSOpen: fsembed.EmbedFSOpenFunc(func(name string) (fs.File, error) {
		return FontsFS.Open(name)
	}), EmbedFSReadDir: fsembed.EmbedFSReadDirFunc(func(name string) ([]fs.DirEntry, error) {
		for _, pth := range paths {
			if pthl, nml := len(pth), len(name); pthl <= nml && name[:pthl] == pth {
				return FontsFS.ReadDir(name)
			}
		}
		return FontsFS.ReadDir(name)
	}), EmbedFSReadFile: fsembed.EmbedFSReadFileFunc(func(name string) ([]byte, error) {
		return FontsFS.ReadFile(name)
	})}
}
