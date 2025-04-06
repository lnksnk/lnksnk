package embed

import (
	gofs "io/fs"
	"path/filepath"
	"strings"

	"github.com/lnksnk/lnksnk/fs"
	"github.com/lnksnk/lnksnk/iorw"
)

type EmbedFS interface {
	Open(name string) (gofs.File, error)
	ReadDir(name string) ([]gofs.DirEntry, error)
	ReadFile(name string) ([]byte, error)
}

// ImoprtResource
// example embed.ImportResource(mltyfsys, fontsext.FSFonts, ".css", true, "/fontsext", "material", "roboto")
func ImportResource(fsys fs.MultiFileSystem, emdfs EmbedFS, srchdrexts string, excldeexts string, incldsubdirs bool, pathroot string, paths ...string) {
	var cptrdpths = map[string][]string{}
	var chksrchdrexts = map[string]bool{}
	if srchdrexts = strings.TrimFunc(srchdrexts, iorw.IsSpace); srchdrexts != "" {
		for _, srcext := range strings.Split(srchdrexts, ",") {
			if srcext = filepath.Ext(srcext); srcext != "" {
				chksrchdrexts[srcext] = true
			}
		}
	}

	var excldexts = map[string]bool{}
	if excldeexts = strings.TrimFunc(excldeexts, iorw.IsSpace); excldeexts != "" {
		for _, ecldext := range strings.Split(excldeexts, ",") {
			if ecldext = filepath.Ext(ecldext); ecldext != "" {
				excldexts[ecldext] = true
			}
		}
	}

	for _, pth := range paths {
		importResourcePath(fsys, emdfs, incldsubdirs, pathroot+"/"+pth, pth, func(epath string, de gofs.DirEntry) (exclde bool) {
			if len(excldeexts) > 0 && !de.IsDir() {
				exclde = excldexts[filepath.Ext(epath)]
			}
			return
		}, func(flpath string, flext string) {
			if flext != "" {
				cptrdpths[flext] = append(cptrdpths[flext], flpath)
			}
		})
	}
	if len(cptrdpths) > 0 {
		srcrf := iorw.NewBuffer()
		for ext := range chksrchdrexts {
			if (srchdrexts == "" && len(chksrchdrexts) == 0) || (srchdrexts != "" && len(chksrchdrexts) > 0 && chksrchdrexts[ext]) {
				pths := cptrdpths[ext]
				if strings.EqualFold(".css", ext) {
					for _, pth := range pths {
						srcrf.Println(`<link rel="stylesheet" type="text/css" href="` + pth + `">`)
					}
					continue
				}
				if strings.EqualFold(".js", ext) {
					for _, pth := range pths {
						srcrf.Println(`<script type="text/javascript" src="` + pth + `"></script>`)
					}
					continue
				}
			}
		}
		if !srcrf.Empty() {
			defer srcrf.Close()
			fsys.Map(pathroot)
			if fi := fsys.Stat(pathroot + "/index.html"); fi == nil {
				fsys.Set(pathroot+"/index.html", srcrf.Reader())
				return
			}
			if fi := fsys.Stat(pathroot + "/head.html"); fi == nil {
				fsys.Set(pathroot+"/head.html", srcrf.Reader())
				return
			}
		}
	}
}

func importResourcePath(fsys fs.MultiFileSystem, emdfs EmbedFS, incldsubdirs bool, pathroot string, path string, excldfl func(string, gofs.DirEntry) bool, cptrdfle func(string, string)) {
	emddirs, _ := emdfs.ReadDir(path)
	if len(emddirs) > 0 {
		fsys.Map(pathroot)
		if path[len(path)-1] != '/' {
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
						cptrdfle(subroot, filepath.Ext(subroot))
					}
				}()
			}
		}
		return
	}
}
