package mimes

import (
	"context"
	_ "embed"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lnksnk/lnksnk/ioext"
)

//go:embed mimetypes.txt
var mimetypescsv string

// MimeTypesCSV - return Mime Types CSV reader
var mimebuf = ioext.NewBuffer()
var mimebuflck = &sync.RWMutex{}

func MimeTypesCSV() *ioext.BuffReader {
	if mimebuf.Size() == 0 {
		func() {
			mimebuflck.Lock()
			defer mimebuflck.Unlock()
			if mimebuf.Size() == 0 {
				mimebuf.Print(mimetypescsv)
			}
		}()
	}
	return mimebuf.Reader().DisposeEOFReader()
}

func ExtMimeType(ext string, defaultext string, defaulttype ...string) (mimetype string) {
	var defaulttpe = ""
	if len(defaulttype) > 0 {
		defaulttpe = defaulttype[0]
	}
	if ext = filepath.Ext(ext); ext == "" {
		ext = filepath.Ext(defaultext)
	}
	mimetype, _, _ = FindMimeType(ext, defaulttpe)
	return
}

// FindMimeType - ext or defaulttype
func FindMimeType(ext string, defaulttpe ...string) (mimetype string, texttype bool, mediatype bool) {
	defaulttype := "text/plain"
	if len(defaulttpe) > 0 && defaulttpe[0] != "" {
		defaulttype = defaulttpe[0]
	}
	texttype = false
	if ext = filepath.Ext(ext); ext != "" {
		func() {
			if mimetpev, mimetypeok := mtypesfound.Load(ext); mimetypeok {
				mimetype, _ = mimetpev.(string)
				if textextv, textextok := mtextexts.Load(ext); textextok {
					texttype, _ = textextv.(bool)
				}
			} else {
				ctx, ctxcancel := context.WithCancel(context.Background())
				go func() {
					defer ctxcancel()
					var bufr = MimeTypesCSV()
					for {
						lineb, lineberr := bufr.Readln()
						if len(lineb) > 0 {
							var lines = strings.Split(string(lineb), "\t")
							if len(lines) == 4 && lines[2] == ext {
								mimetype = lines[1]
								mtypesfound.Store(ext, mimetype)
								if textextv, textextok := mtextexts.Load(ext); textextok {
									texttype, _ = textextv.(bool)
								}
								break
							}
						}
						if lineberr != nil {
							break
						}
					}
					bufr = nil
				}()
				<-ctx.Done()
				if mimetype == "" {
					if mimetype = defaulttype; mimetype == "" {
						mimetype = "text/plain"
					}
				}
			}
		}()
	} else {
		mimetype = defaulttype
	}
	mediatype = strings.Contains(mimetype, "video/") || strings.Contains(mimetype, "audio/")
	return
}

var mtypesfound = &sync.Map{}
var mtextexts = &sync.Map{}

func init() {
	mtypesfound = &sync.Map{}
	mtextexts.Store(".js", true)
	mtextexts.Store(".json", true)
	mtextexts.Store(".html", true)
	mtextexts.Store(".xhtml", true)
	mtextexts.Store(".htm", true)
	mtextexts.Store(".js", true)
}
