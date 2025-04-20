package embed

import (
	gofs "io/fs"
	"path/filepath"
	"strings"

	"github.com/lnksnk/lnksnk/fs"
	"github.com/lnksnk/lnksnk/ioext"
)

type EmbedFS interface {
	Open(name string) (gofs.File, error)
	ReadDir(name string) ([]gofs.DirEntry, error)
	ReadFile(name string) ([]byte, error)
}

type EmbedFSReadFile interface {
	ReadFile(name string) ([]byte, error)
}

type EmbedFSReadFileFunc func(string) ([]byte, error)

func (emdfsreadfile EmbedFSReadFileFunc) ReadFile(name string) ([]byte, error) {
	return emdfsreadfile(name)
}

type EmbedFSReadDir interface {
	ReadDir(name string) ([]gofs.DirEntry, error)
}

type EmbedFSReadDirFunc func(string) ([]gofs.DirEntry, error)

func (emdfsreaddir EmbedFSReadDirFunc) ReadDir(name string) ([]gofs.DirEntry, error) {
	return emdfsreaddir(name)
}

type EmbedFSOpen interface {
	Open(string) (gofs.File, error)
}

type EmbedFSOpenFunc func(string) (gofs.File, error)

func (emdfsopen EmbedFSOpenFunc) Open(name string) (gofs.File, error) {
	return emdfsopen(name)
}

// ImoprtResource
// example embed.ImportResource(mltyfsys, fontsext.FSFonts, ".css", true, "/fontsext", "material", "roboto")
func ImportResource(capturedsource func(srcroot string, src *ioext.Buffer, srcfsys fs.MultiFileSystem), fsys fs.MultiFileSystem, emdfs EmbedFS, srchdrexts string, excldeexts string, incldsubdirs bool, pathroot string, paths ...string) {
	var cptrdpths = map[string][]string{}
	var chksrchdrexts = map[string]bool{}
	if srchdrexts = strings.TrimFunc(srchdrexts, ioext.IsSpace); srchdrexts != "" {
		for _, srcext := range strings.Split(srchdrexts, ",") {
			tstext := ""
			for {
				tmpext := filepath.Ext(srcext[:len(srcext)-len(tstext)])
				if tmpext == "" {
					break
				}
				tstext = tmpext + tstext
			}
			if tstext != "" {
				chksrchdrexts[tstext] = true
			}
		}
	}

	var excldexts = map[string]bool{}
	if excldeexts = strings.TrimFunc(excldeexts, ioext.IsSpace); excldeexts != "" {
		for _, ecldext := range strings.Split(excldeexts, ",") {
			if ecldext = strings.TrimFunc(ecldext, ioext.IsSpace); ecldext != "" {
				tstext := ""

				for {
					tmpext := filepath.Ext(ecldext[:len(ecldext)-len(tstext)])
					if tmpext == "" {
						break
					}
					tstext = tmpext + tstext
				}
				if tstext != "" {
					excldexts[tstext] = true
				}
			}
		}
	}

	for _, pth := range paths {
		if pathroot != "" {
			if pathroot == "/" {
				pathroot = ""
			} else {
				if pathroot[len(pathroot)-1] == '/' {
					pathroot = pathroot[:len(pathroot)-1]
				}
			}

		}
		importResourcePath(fsys, emdfs, incldsubdirs, pathroot+func() string {
			if pth != "" {
				if pth[0] != '/' && pathroot[len(pathroot)-1] != '/' {
					return "/" + pth
				}
				if pth[0] == '/' && pathroot[len(pathroot)-1] != '/' {
					return pth
				}
				if pth[0] != '/' && pathroot[len(pathroot)-1] == '/' {
					return pth
				}
			}
			return pth
		}(), pth, func(epath string, de gofs.DirEntry) (exclde bool) {
			if len(excldexts) > 0 && !de.IsDir() {
				tstext := ""
				for {
					tmpext := filepath.Ext(epath[:len(epath)-len(tstext)])
					if tmpext == "" {
						break
					}
					tstext = tmpext + tstext
				}

				exclde = tstext != "" && excldexts[tstext]
			}
			return
		}, func(flpath string, flext string) {
			if flext != "" {
				cptrdpths[flext] = append(cptrdpths[flext], flpath)
			}
		})
	}
	if len(cptrdpths) > 0 {
		srcrf := ioext.NewBuffer()
		for ext := range chksrchdrexts {
			if (srchdrexts == "" && len(chksrchdrexts) == 0) || (srchdrexts != "" && len(chksrchdrexts) > 0 && chksrchdrexts[ext]) {
				pths := cptrdpths[ext]
				if strings.EqualFold(".css", ext) || strings.EqualFold(".min.css", ext) {
					for _, pth := range pths {
						srcrf.Println(`<link rel="stylesheet" type="text/css" href="` + pth + `">`)
					}
					continue
				}
				if strings.EqualFold(".js", ext) || strings.EqualFold(".min.js", ext) {
					for _, pth := range pths {
						srcrf.Println(`<script type="text/javascript" src="` + pth + `"></script>`)
					}
					continue
				}
			}
		}
		if !srcrf.Empty() {
			defer srcrf.Close()
			if capturedsource == nil {
				fsys.Map(pathroot)
				if !fsys.Exist(pathroot + "/index.html") {
					fsys.Set(pathroot+"/index.html", srcrf.Reader())
					return
				}
				return
			}
			capturedsource(pathroot, srcrf, fsys)
		}
	}
}

func importResourcePath(fsys fs.MultiFileSystem, emdfs EmbedFS, incldsubdirs bool, pathroot string, path string, excldfl func(string, gofs.DirEntry) bool, cptrdfle func(string, string)) {
	emddirs, _ := emdfs.ReadDir(func() string {
		if path == "" {
			return "."
		}
		return path
	}())
	if len(emddirs) > 0 {
		fsys.Map(pathroot)
		if path == "" || path[len(path)-1] != '/' {
			path += "/"
		}
		if path[0] != '/' {
			path = "/" + path
		}
		for _, edr := range emddirs {
			if edr.IsDir() {
				subroot := pathroot

				if subroot[len(subroot)-1] == '/' {
					subroot += edr.Name()
				} else {
					subroot += "/" + edr.Name()
				}
				dirpth := path
				if dirpth[len(dirpth)-1] == '/' {
					dirpth += edr.Name()
				} else {
					dirpth += "/" + edr.Name()
				}
				fsys.Map(subroot)
				if dirpth[0] == '/' {
					dirpth = dirpth[1:]
				}
				if excldfl != nil && excldfl(subroot, edr) {
					continue
				}
				importResourcePath(fsys, emdfs, incldsubdirs, subroot, dirpth, excldfl, cptrdfle)
				continue
			}

			subroot := pathroot
			subroot += "/" + edr.Name()
			dirpth := path
			if dirpth[len(dirpth)-1] == '/' {
				dirpth += edr.Name()
			} else {
				dirpth += "/" + edr.Name()
			}
			if dirpth[0] == '/' {
				dirpth = dirpth[1:]
			}
			if excldfl != nil && excldfl(subroot, edr) {
				continue
			}
			if f, _ := emdfs.Open(dirpth); f != nil {
				func() {
					defer f.Close()
					fsys.Set(subroot, f)
					if cptrdfle != nil {
						cptrext := ""
						for {
							cptrtmpext := filepath.Ext(subroot[:len(subroot)-len(cptrext)])
							if cptrtmpext == "" {
								break
							}
							cptrext = cptrtmpext + cptrext
						}
						cptrdfle(subroot, cptrext)
					}
				}()
			}
		}
		return
	}
}
