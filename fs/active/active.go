package active

import (
	"context"
	"fmt"
	"io"
	gofs "io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/lnksnk/lnksnk/fs"
	"github.com/lnksnk/lnksnk/ioext"
	"github.com/lnksnk/lnksnk/template"
)

type CachedInfo interface {
	fs.FileInfo
	Content() io.Reader
	ContentSize() int64
	Code() io.Reader
	CodeSize() int64
	Program() interface{}
	Close() error
	Valid() bool
	Plain() bool
}

type CachedInfos interface {
	ioext.IterateMap[string, CachedInfo]
}

type chachedinfos struct {
	*ioext.MapIterateEvents[string, CachedInfo]
	CachedInfos
}

func NewCachedInfos(chdmp map[string]CachedInfo) (chdinfos *chachedinfos) {
	chdinfos = &chachedinfos{CachedInfos: ioext.MapIterator[string, CachedInfo](), MapIterateEvents: &ioext.MapIterateEvents[string, CachedInfo]{}}
	return
}

func CachedEvents(chdinfos *chachedinfos) *ioext.MapIterateEvents[string, CachedInfo] {
	if chdinfos == nil {
		return nil
	}
	return chdinfos.MapIterateEvents
}

func (chdinfos *chachedinfos) Get(name string) (CachedInfo, bool) {
	if chdinfos == nil || chdinfos.CachedInfos == nil {
		return nil, false
	}
	return chdinfos.CachedInfos.Get(name)
}

type activeFileSystem struct {
	fs.MultiFileSystem
	chdfis CachedInfos
	cmple  func(fsys fs.MultiFileSystem, cde ...interface{}) (prgm interface{}, err error)
}

type cachedinfo struct {
	//chdinfos *chachedinfos
	finfos map[string]fs.FileInfo
	*fileinfo
	unmtchd map[string]bool
	code    *ioext.Buffer
	cntnt   *ioext.Buffer
	prgm    interface{}
}

// Plain implements CachedInfo.
func (c *cachedinfo) Plain() bool {
	if c == nil {
		return false
	}
	return len(c.finfos) == 0 && c.code.Empty()
}

// Valid implements CachedInfo.
func (c *cachedinfo) Valid() bool {
	if c == nil {
		return false
	}
	for _, tc := range c.finfos {
		if tcc, _ := tc.(CachedInfo); tcc != nil && !tcc.Valid() {
			return false
		}
	}
	return true
}

// Active implements CachedInfo.
// Subtle: this method shadows the method (*fileinfo).Active of cachedinfo.fileinfo.
func (c *cachedinfo) Active() bool {
	return true
}

// Base implements CachedInfo.
// Subtle: this method shadows the method (*fileinfo).Base of cachedinfo.fileinfo.
func (c *cachedinfo) Base() string {
	return c.base
}

// Code implements CachedInfo.
func (c *cachedinfo) Code() io.Reader {
	return c.code.Reader()
}

// CodeSize implements CachedInfo.
func (c *cachedinfo) CodeSize() int64 {
	return c.code.Size()
}

// Ext implements CachedInfo.
// Subtle: this method shadows the method (*fileinfo).Ext of cachedinfo.fileinfo.
func (c *cachedinfo) Ext() string {
	return c.ext
}

// IsDir implements CachedInfo.
// Subtle: this method shadows the method (*fileinfo).IsDir of cachedinfo.fileinfo.
func (c *cachedinfo) IsDir() bool {
	return false
}

// Media implements CachedInfo.
// Subtle: this method shadows the method (*fileinfo).Media of cachedinfo.fileinfo.
func (c *cachedinfo) Media() bool {
	return false
}

// ModTime implements CachedInfo.
// Subtle: this method shadows the method (*fileinfo).ModTime of cachedinfo.fileinfo.
func (c *cachedinfo) ModTime() time.Time {
	return c.modtime
}

// Mode implements CachedInfo.
// Subtle: this method shadows the method (*fileinfo).Mode of cachedinfo.fileinfo.
func (c *cachedinfo) Mode() gofs.FileMode {
	return c.mod
}

// Name implements CachedInfo.
// Subtle: this method shadows the method (*fileinfo).Name of cachedinfo.fileinfo.
func (c *cachedinfo) Name() string {
	return c.name
}

// Content implements CachedInfo.
func (c *cachedinfo) Content() io.Reader {
	return c.cntnt.Reader()
}

// ContentSize implements CachedInfo.
func (c *cachedinfo) ContentSize() int64 {
	return c.cntnt.Size()
}

// Path implements CachedInfo.
// Subtle: this method shadows the method (*fileinfo).Path of cachedinfo.fileinfo.
func (c *cachedinfo) Path() string {
	return c.path
}

// Program implements CachedInfo.
func (c *cachedinfo) Program() interface{} {
	return c.prgm
}

// Raw implements CachedInfo.
// Subtle: this method shadows the method (*fileinfo).Raw of cachedinfo.fileinfo.
func (c *cachedinfo) Raw() bool {
	return false
}

// Reader implements CachedInfo.
// Subtle: this method shadows the method (*fileinfo).Reader of cachedinfo.fileinfo.
func (c *cachedinfo) Reader(ctx ...context.Context) io.Reader {
	return c.cntnt.Reader(func() context.Context {
		if len(ctx) > 0 {
			return ctx[0]
		}
		return nil
	}())
}

// Root implements CachedInfo.
// Subtle: this method shadows the method (*fileinfo).Root of cachedinfo.fileinfo.
func (c *cachedinfo) Root() string {
	return c.root
}

// Size implements CachedInfo.
// Subtle: this method shadows the method (*fileinfo).Size of cachedinfo.fileinfo.
func (c *cachedinfo) Size() int64 {
	return c.cntnt.Size()
}

// Sys implements CachedInfo.
// Subtle: this method shadows the method (*fileinfo).Sys of cachedinfo.fileinfo.
func (c *cachedinfo) Sys() any {
	return nil
}

func (c *cachedinfo) Close() (err error) {
	if c != nil {
		fileinfo := c.fileinfo
		c.fileinfo = nil
		c.cntnt.Close()
		c.cntnt = nil
		c.code.Close()
		c.code = nil
		c.prgm = nil
		if fileinfo != nil {
			buffer := fileinfo.Buffer
			fileinfo.Buffer = nil
			if buffer != nil {
				buffer.Close()
			}
		}
	}
	return
}

type fileinfo struct {
	*ioext.Buffer
	ext     string
	path    string
	base    string
	root    string
	name    string
	modtime time.Time
	isdir   bool
	mod     gofs.FileMode
}

func fileInfoFromFi(fi fs.FileInfo) (chdfi *fileinfo) {
	if fi != nil {
		chdfi = &fileinfo{
			Buffer:  ioext.NewBuffer(fi.Reader()),
			ext:     fi.Ext(),
			path:    fi.Path(),
			base:    fi.Base(),
			root:    fi.Root(),
			name:    fi.Name(),
			modtime: fi.ModTime(),
			mod:     fi.Mode(),
			isdir:   fi.IsDir()}
	}
	return
}

func (chdfi *fileinfo) Active() bool {
	return true
}

func (chdfi *fileinfo) Raw() bool {
	return false
}

func (chdfi *fileinfo) Media() bool {
	return false
}

func (chdfi *fileinfo) Ext() string {
	if chdfi == nil {
		return ""
	}
	return chdfi.ext
}

func (chdfi *fileinfo) Path() string {
	if chdfi == nil {
		return ""
	}
	return chdfi.path
}

func (chdfi *fileinfo) Name() string {
	if chdfi == nil {
		return ""
	}
	return chdfi.name
}

func (chdfi *fileinfo) Size() int64 {
	if chdfi == nil {
		return 0
	}
	return chdfi.Buffer.Size()
}

func (chdfi *fileinfo) Mode() gofs.FileMode {
	if chdfi == nil {
		return 0
	}
	return chdfi.mod
}

func (chdfi *fileinfo) ModTime() time.Time {
	if chdfi == nil {
		return time.Now()
	}
	return chdfi.modtime
}

func (chdfi *fileinfo) IsDir() bool {
	if chdfi == nil {
		return false
	}
	return chdfi.isdir
}

func (chdfi *fileinfo) Sys() any {
	return nil
}

func (chdfi *fileinfo) Root() string {
	if chdfi == nil {
		return ""
	}
	return chdfi.root
}

func (chdfi *fileinfo) Base() string {
	if chdfi == nil {
		return ""
	}
	return chdfi.base
}

func (chdfi *fileinfo) Reader(ctx ...context.Context) io.Reader {
	if chdfi == nil {
		return nil
	}
	if len(ctx) > 0 {
		return chdfi.Buffer.Reader(ctx[0])
	}
	return chdfi.Buffer.Reader()
}

func Parse(fsys fs.MultiFileSystem, chdinfos CachedInfos, fi fs.FileInfo, out io.Writer, compile func(a ...interface{}) (interface{}, error), run func(interface{}, io.Writer)) (chdinfo CachedInfo) {
	if chdinfos == nil {
		return
	}
	tstpath := fi.Path()
	//tstpath += fi.Path()[1:]
	chkfnd := false
	if chdinfo, chkfnd = chdinfos.Get(tstpath); chkfnd && chdinfo != nil {
		if chdinfo.ModTime() == fi.ModTime() {
			chdpsv := chdinfo.Content()
			//chdatv := chdinfo.Code()
			go func(reffsys fs.MultiFileSystem, reffi fs.FileInfo, chdfios CachedInfos, chdfi CachedInfo) {
				if !chdfi.Valid() {
					reftstpath := chdinfo.Path()
					chdfios.Delete(reftstpath)
					createcachedfileinfo(chdfios, reffi, reffsys, compile)
				}
			}(fsys, fi, chdinfos, chdinfo)
			if out != nil {
				ioext.Fprint(out, chdpsv)
			}
			if !chdinfo.Plain() && run != nil {
				if chdprgm := chdinfo.Program(); chdprgm != nil {
					run(chdprgm, out)
				}
			}
			return
		}
		chdinfos.Delete(chdinfo.Path())
		chdinfo.Close()
	}
	if chdinfo = createcachedfileinfo(chdinfos, fi, fsys, compile); chdinfo != nil {
		if out != nil {
			ioext.Fprint(out, chdinfo.Content())
		}
		if pgrm := chdinfo.Program(); pgrm != nil && run != nil {
			run(pgrm, out)
		}
	}
	return
}

func createcachedfileinfo(chdinfos CachedInfos, fi fs.FileInfo, fsys fs.MultiFileSystem, compile func(...interface{}) (interface{}, error)) (chdinfo CachedInfo) {
	chdinfo = &cachedinfo{fileinfo: fileInfoFromFi(fi), finfos: map[string]fs.FileInfo{}}
	internparse(fsys, chdinfos, chdinfo, compile)
	chdinfos.Set(chdinfo.Path(), chdinfo)
	return chdinfo
}

func internsync(fsys fs.MultiFileSystem, chdinfos CachedInfos, path string, unmatched bool, compile func(...interface{}) (interface{}, error)) {
	tstname := strings.Replace(path, "/", ":", -1)
	tstext := filepath.Ext(path)
	if strings.HasSuffix(tstname, "index"+tstext) {
		tstname = tstname[:len(tstname)-len("index"+tstext)]
	} else if strings.HasSuffix(tstname, tstext) {
		tstname = tstname[:len(tstname)-len(tstext)]
	}
	tsnmel := len(tstname)
	if atvfsys, _ := fsys.(*activeFileSystem); atvfsys != nil {
		for _, chdfis := range chdinfos.Iterate() {
			if tstchdfi, _ := chdfis.(*cachedinfo); tstchdfi != nil {
				if unmthd := tstchdfi.unmtchd; len(unmthd) > 0 {
					if bsel := len(tstchdfi.base); bsel < tsnmel && tstname[:bsel] == strings.Replace(tstchdfi.base, "/", ":", -1) && unmthd[tstname[bsel-1:]] {
						if unmatched {
							internparse(fsys, chdinfos, tstchdfi, compile)
						}
					} else if unmthd[tstname] {
						if unmatched {
							internparse(fsys, chdinfos, tstchdfi, compile)
						}
					}
				}
				if finfos := tstchdfi.finfos; len(finfos) > 0 {
					if bsel := len(tstchdfi.base); bsel < tsnmel && tstname[:bsel] == strings.Replace(tstchdfi.base, "/", ":", -1) && finfos[tstname[bsel-1:]] != nil {
						internparse(fsys, chdinfos, tstchdfi, compile)
					} else if finfos[tstname] != nil {
						internparse(fsys, chdinfos, tstchdfi, compile)
					}
				}
			}
		}
	}
}

func internparse(fsys fs.MultiFileSystem, chdinfos CachedInfos, fi CachedInfo, compile func(...interface{}) (interface{}, error)) {
	if chdfi, _ := fi.(*cachedinfo); chdfi != nil {
		mrkptmplt := template.MarkupTemplate(fsys, fi)
		mrkptmplt.Parse(chdfi.Buffer)
		mrkptmplt.Wrapup()
		chdfi.cntnt = mrkptmplt.Content()
		chdfi.code = mrkptmplt.Code()
		chdfi.finfos = mrkptmplt.ValidElements()
		if !chdfi.code.Empty() {
			var err error
			chdfi.prgm, err = compile(chdfi.code)
			if err != nil {
				fmt.Println("err:" + err.Error())
				cdelines := ""
				for i, ln := range strings.Split(chdfi.code.String(), "\n") {
					cdelines += fmt.Sprintf("%d %s\r\n", i+1, strings.TrimRightFunc(ln, ioext.IsSpace))
				}
				fmt.Println(cdelines)
				fmt.Println()
			}
		}
		chdfi.unmtchd = mrkptmplt.InvalidElements()
		internsync(fsys, chdinfos, chdfi.path, true, compile)
		return
	}
}

func (atvfsys *activeFileSystem) CachedInfo(path string) (chdfi CachedInfo, err error) {
	if atvfsys == nil {
		return
	}
	return
}

func (atvfsys *activeFileSystem) Map(path ...interface{}) (fsys fs.FileSystem) {
	if atvfsys == nil || len(path) == 0 {

	}
	path = append(path, func(fs fs.FileSystem, fi fs.FileInfo, ntfy fs.Notify) {
		atvfsys.update(fs, fi, ntfy)
	})
	if mltyfsys := atvfsys.MultiFileSystem; mltyfsys != nil {
		fsys = mltyfsys.Map(path...)
	}
	return
}

func (atvfsys *activeFileSystem) compile(cde ...interface{}) (prgm interface{}, err error) {
	if atvfsys == nil {
		return
	}
	if cmple := atvfsys.cmple; cmple != nil {
		return cmple(atvfsys, cde...)
	}
	return
}

func (atvfsys *activeFileSystem) update(fsys fs.FileSystem, fi fs.FileInfo, ntfy fs.Notify) {
	if atvfsys == nil {
		return
	}
	if fsys != nil && fi != nil && fi.Active() {
		if chcdfis := atvfsys.chdfis; chcdfis != nil {

			chdfi, chdfiok := chcdfis.Get(fi.Path())
			if ntfy == fs.NOTE_AMMEND {
				if chdfiok {
					chdfi.Close()
					chcdfis.Delete(fi.Path())
				}
				Parse(atvfsys, atvfsys.chdfis, fi, nil, atvfsys.compile, nil)
				return
			}
			if ntfy == fs.NOTE_CREATE {
				if chdfiok {
					chdfi.Close()
					chcdfis.Delete(fi.Path())
				}
				Parse(atvfsys, atvfsys.chdfis, fi, nil, atvfsys.compile, nil)
				return
			}
			if ntfy == fs.NOTE_REMOVE {
				if chdfiok {
					func() {
						defer chdfi.Close()
						chcdfis.Delete(fi.Path())
						internsync(atvfsys, atvfsys.chdfis, chdfi.Path(), false, atvfsys.compile)
					}()

				}
			}
		}
	}
}

func AciveFileSystem(compile func(fsys fs.MultiFileSystem, cde ...interface{}) (prgm interface{}, err error), mltyfsys ...fs.MultiFileSystem) (actvmltifsys *activeFileSystem) {
	if len(mltyfsys) == 0 {
		if compile == nil {
			compile = func(fs.MultiFileSystem, ...interface{}) (prgm interface{}, err error) {
				return
			}
		}
		actvmltifsys = &activeFileSystem{MultiFileSystem: fs.NewMultiFileSystem(), chdfis: NewCachedInfos(nil), cmple: compile}
		if events, _ := actvmltifsys.chdfis.Events().(*ioext.MapIterateEvents[string, CachedInfo]); events != nil {

		}
		actvmltifsys.Notifier(func(fs fs.FileSystem, fi fs.FileInfo, ntfy fs.Notify) {
			if fi.Active() {
				actvmltifsys.update(fs, fi, ntfy)
			}
		}, fs.NOTE_AMMEND, fs.NOTE_CREATE, fs.NOTE_REMOVE)
	}
	return
}

func ProcessActiveFile(mltyfsys fs.MultiFileSystem, fi fs.FileInfo, out io.Writer, compile func(a ...interface{}) (interface{}, error), run func(interface{}, io.Writer)) fs.FileInfo {
	if fi != nil && fi.Active() {
		if atvfsys, _ := mltyfsys.(*activeFileSystem); atvfsys != nil {
			if compile == nil {
				compile = atvfsys.compile
			}
			fi = Parse(atvfsys, atvfsys.chdfis, fi, out, compile, run)
		}
		return nil
	}
	return fi
}
func init() {

}
