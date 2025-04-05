package fs

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/lnksnk/lnksnk/iorw"
)

type MultiFileSystem interface {
	Open(string) File
	OpenContext(context.Context, string) File
	List(string) []FileInfo
	StatContext(context.Context, string) FileInfo
	Stat(string) FileInfo
	Exist(string) bool
	Map(...interface{}) FileSystem
	CacheExtensions(...string)
	ActiveExtensions(...string)
	DefaultExtensions(...string)
	Set(string, ...interface{}) bool
	Iterate(...string) func(func(string, FileSystem) bool)
	Notifier(func(FileSystem, FileInfo, Notify), ...Notify)
	Notify(FileSystem, FileInfo, Notify)
}

type Notify int

func (ntfy Notify) String() string {
	if ntfy == NOTE_AMMEND {
		return "changed"
	}
	if ntfy == NOTE_CREATE {
		return "created"
	}
	if ntfy == NOTE_REMOVE {
		return "removed"
	}
	return "unknown"
}

const (
	NOTE_UKNOWN Notify = iota
	NOTE_CREATE
	NOTE_AMMEND
	NOTE_REMOVE
)

type multifilesys struct {
	ntyfiers  map[Notify]func(FileSystem, FileInfo, Notify)
	fsystms   map[string]FileSystem
	wtchdfsys map[string]FileSystem
	wtchr     *watcher
	chdexts   map[string]bool
	actvexts  map[string]bool
	dfltexts  map[string]bool
}

// Notify implements MultiFileSystem.
func (mltyfsys *multifilesys) Notify(fss FileSystem, fi FileInfo, ntfy Notify) {
	if mltyfsys == nil {
		return
	}
	if ntfycaller := mltyfsys.ntyfiers[ntfy]; ntfycaller != nil {
		ntfycaller(fss, fi, ntfy)
		return
	}
}

// Set implements MultiFileSystem.
func (mltyfsys *multifilesys) Set(path string, a ...interface{}) bool {
	if mltyfsys == nil || path == "" {
		return false
	}
	if path[0] != '/' {
		path = "/" + path
	}
	if fsystms := mltyfsys.fsystms; fsystms != nil {
		root := path[:strings.LastIndex(path, "/")+1]
		if root != "/" && root[0] == '/' {
			if root[len(root)-1] == '/' {
				root = root[:len(root)-1]
			}
		}
		var fsys FileSystem = fsystms[root]
		if fsys != nil {
			path = path[len(root):]
			return fsys.Set(path, a...)
		}
		rtl := len(root)
		for n := range rtl {
			if root[rtl-(n+1)] == '/' {
				if fsys = fsystms[root[:rtl-(n+1)]]; fsys != nil {
					root = root[:rtl-(n+1)]
					path = path[len(root):]
					return fsys.Set(path, a...)
				}
			}
		}
	}
	return false
}

// Notifier implements MultiFileSystem.
func (mltyfsys *multifilesys) Notifier(ntfyfunc func(FileSystem, FileInfo, Notify), ntfy ...Notify) {
	if mltyfsys == nil {
		return
	}
	if ntyfiers := mltyfsys.ntyfiers; ntyfiers != nil && ntfyfunc != nil {
		for _, nfy := range ntfy {
			ntyfiers[nfy] = ntfyfunc
		}
	}
}

// StatContext implements MultiFileSystem.
func (mltyfsys *multifilesys) StatContext(ctx context.Context, path string) (fi FileInfo) {
	if mltyfsys == nil || path == "" {
		return nil
	}
	if ctx != nil {
		if ctx.Err() != nil {
			return
		}
	}
	fsystms := mltyfsys.fsystms
	pthl := len(path)
	pn := 0
	for pn < pthl {
		if path[pn] == '/' {
		retry:
			if fsfnd := fsystms[path[:func() int {
				if pn == 0 {
					return 1
				}
				return pn
			}()]]; fsfnd != nil {
				if fi = fsfnd.StatContext(ctx, path[pn:]); fi != nil {
					return
				}
			}

			if pn < pthl-4 && path[pn:pn+4] == "/../" {
			strip:
				path = path[:pn+1] + path[pn+4:]
				pthl -= 4
				if pn < pthl-4 && path[pn:pn+4] == "/../" {
					goto strip
				}
				goto retry
			}
		}
		if tpn := pthl - (pn + 1); tpn > pn && path[tpn] == '/' {
		retrytpn:
			if tpn > 3 && path[tpn-3:tpn+1] == "/../" {
				tpn -= 3
				goto retrytpn
			}
			if fsfnd := fsystms[path[:tpn]]; fsfnd != nil {
				if fi = fsfnd.StatContext(ctx, path[tpn:]); fi != nil {
					return
				}
			}
			for tpn > pn {
				tpn--
				if tpn == pn {
					pn = pthl - 1
					break
				}
				if path[tpn] == '/' {
					goto retrytpn
				}
			}
		}
		pn++
	}

	return
}

// Stat implements MultiFileSystem.
func (mltyfsys *multifilesys) Stat(path string) (fi FileInfo) {
	return mltyfsys.StatContext(context.Background(), path)
}

// Iterate implements MultiFileSystem.
func (mltyfsys *multifilesys) Iterate(syspaths ...string) func(func(string, FileSystem) bool) {
	return func(yield func(string, FileSystem) bool) {
		if mltyfsys == nil {
			return
		}
		if fsystms := mltyfsys.fsystms; fsystms != nil {
			if len(syspaths) > 0 {
				spths := []string{}
				spthsmp := map[string]FileSystem{}
				for _, spth := range syspaths {
					if fs, _ := mltyfsys.fsystms[spth]; fs != nil {
						if spthsmp[spth] == nil {
							spths = append(spths, spth)
							spthsmp[spth] = fs
						}
					}
				}
				for _, spth := range spths {
					if !yield(spth, spthsmp[spth]) {
						return
					}
				}
				return
			}
			for key, value := range fsystms {
				if !yield(key, value) {
					return
				}
			}
		}
	}
}

// CacheExtensions implements MultiFileSystem.
func (mltyfsys *multifilesys) CacheExtensions(exts ...string) {
	if mltyfsys == nil && len(exts) == 0 {
		return
	}
	if chdexts := mltyfsys.chdexts; chdexts != nil {
		for _, ext := range exts {
			if ext = filepath.Ext(ext); ext != "" {
				if !chdexts[ext] {
					chdexts[ext] = true
				}
			}
		}
	}
}

// ActiveExtensions implements MultiFileSystem.
func (mltyfsys *multifilesys) ActiveExtensions(exts ...string) {
	if mltyfsys == nil && len(exts) == 0 {
		return
	}
	if actvexts := mltyfsys.actvexts; actvexts != nil {
		for _, ext := range exts {
			if ext = filepath.Ext(ext); ext != "" {
				if !actvexts[ext] {
					actvexts[ext] = true
				}
			}
		}
	}
}

// DefaultExtensions implements MultiFileSystem.
func (mltyfsys *multifilesys) DefaultExtensions(exts ...string) {
	if mltyfsys == nil && len(exts) == 0 {
		return
	}
	if dfltexts := mltyfsys.dfltexts; dfltexts != nil {
		for _, ext := range exts {
			if ext = filepath.Ext(ext); ext != "" {
				if !dfltexts[ext] {
					dfltexts[ext] = true
				}
			}
		}
	}
}

// Exist implements MultiFileSystem.
func (mltyfsys *multifilesys) Exist(string) bool {
	panic("unimplemented")
}

// List implements MultiFileSystem.
func (mltyfsys *multifilesys) List(string) []FileInfo {
	panic("unimplemented")
}

func (mltyfsys *multifilesys) Map(path ...interface{}) (fsys FileSystem) {
	if mltyfsys == nil {
		return
	}
	var ntfyfunc func(FileInfo, Notify)
	var ntfyfsysfunc func(FileSystem, FileInfo, Notify)
	pthl := len(path)
	pthi := 0
	async := false
	for pthi < pthl {
		if ps, psok := path[pthi].(string); psok {
			if ps == "" {
				path = append(path[:pthi], path[pthi+1:]...)
				pthl--
				continue
			}
			pthi++
			continue
		}
		if ntfyfuncd, ntfyfuncdok := path[pthi].(func(FileInfo, Notify)); ntfyfuncdok {
			if ntfyfuncd != nil {
				if ntfyfunc == nil {
					ntfyfunc = ntfyfuncd
				}
			}
			path = append(path[:pthi], path[pthi+1:]...)
			pthl--
			continue
		}
		if ntfyfsysfuncd, ntfyfsysfuncdok := path[pthi].(func(FileSystem, FileInfo, Notify)); ntfyfsysfuncdok {
			if ntfyfsysfuncd != nil {
				if ntfyfsysfunc == nil {
					ntfyfsysfunc = ntfyfsysfuncd
				}
			}
			path = append(path[:pthi], path[pthi+1:]...)
			pthl--
			continue
		}
		if pb, pbok := path[pthi].(bool); pbok {
			async = pb
			path = append(path[:pthi], path[pthi+1:]...)
			pthl--
			continue
		}
		path = append(path[:pthi], path[pthi+1:]...)
		pthl--
	}
	if len(path) == 0 || path[0] == "" {
		return
	}
	if fsystms := mltyfsys.fsystms; fsystms != nil {
		if fsys = fsystms[path[0].(string)]; fsys != nil {
			return
		}
		if fsys = NewFileSystem(func() string {
			if len(path) <= 1 {
				return ""
			}
			return path[1].(string)
		}()); fsys != nil {
			if ntfyfsysfunc != nil && ntfyfunc == nil {
				ntfyfunc = func(fi FileInfo, n Notify) {
					ntfyfsysfunc(fsys, fi, n)
				}
			}
			if fss, _ := fsys.(*filesys); fss != nil {
				fsystms[path[0].(string)] = fss
				fss.mltyfsys = mltyfsys
				fss.mltypath = path[0].(string)
				if fss.mltypath != "" && fss.mltypath[len(fss.mltypath)-1] != '/' {
					fss.mltypath += "/"
				}
				for k, v := range mltyfsys.actvexts {
					fss.activexts[k] = v
				}
				for k, v := range mltyfsys.chdexts {
					fss.cachexts[k] = v
				}
				for k, v := range mltyfsys.dfltexts {
					fss.defaultexts[k] = v
				}
				if async && pthl == 2 {
					mltyfsys.AutoSync(path[pthl-1].(string), fsys)
					if ntfyfunc != nil {
						fsys.AutoSync("/", ntfyfunc)
					} else {
						fsys.AutoSync("/")
					}
				}
			} else {
				fsystms[path[0].(string)] = fsys
			}
		}
	}
	return
}

func NewMultiFileSystem() MultiFileSystem {
	return &multifilesys{ntyfiers: map[Notify]func(FileSystem, FileInfo, Notify){}, fsystms: map[string]FileSystem{}, chdexts: map[string]bool{}, actvexts: map[string]bool{}, dfltexts: map[string]bool{}}
}

func (mltyfsys *multifilesys) AutoSync(path string, fsys FileSystem) {
	if mltyfsys == nil && path == "" {
		return
	}

	wtchr := mltyfsys.wtchr
	trvfsys := func(pth string) (fsys FileSystem) {
		if wtchdfsys := mltyfsys.wtchdfsys; len(wtchdfsys) > 0 {
			for wtchpth, wtchfs := range wtchdfsys {
				if pthl, wthl := len(pth), len(wtchpth); pthl >= wthl && wtchpth[:wthl] == pth[:wthl] {
					fsys = wtchfs
					return
				}
			}
		}
		return
	}
	if wtchr == nil {
		mltyfsys.wtchr = invokeWatcher(func(path string) {
			mltyfsys.EventCreate(path, trvfsys(path))
		}, func(path string) {
			mltyfsys.EventRename(path, trvfsys(path))
		}, func(path string) {
			mltyfsys.EventRemove(path, trvfsys(path))
		}, func(path string) {
			mltyfsys.EventWrite(path, trvfsys(path))
		}, func(err error) {
			mltyfsys.EventError(err, trvfsys(path))
		})
		wtchr = mltyfsys.wtchr
	}
	wtchdfsys := mltyfsys.wtchdfsys
	if wtchdfsys == nil {
		mltyfsys.wtchdfsys = map[string]FileSystem{}
		wtchdfsys = mltyfsys.wtchdfsys
	}
	if wtchr.Add(path) {
		wtchdfsys[path] = fsys
	}
}

func (mltyfsys *multifilesys) EventError(err error, fsys FileSystem) {
	if fsys != nil {
		fmt.Println("Error:" + err.Error())
	}
}

func (mltyfsys *multifilesys) EventRemove(path string, fsys FileSystem) {
	if fsys != nil {
		if fss, _ := fsys.(*filesys); fss != nil {
			if path = path[len(fss.root):]; path != "" {
				fss.Remove(path, func(fi FileInfo) {
					mltyfsys.Notify(fss, fi, NOTE_REMOVE)
				})
			}
		}
	}
}

func (mltyfsys *multifilesys) EventRename(path string, fsys FileSystem) {
	if fsys != nil {
		if fss, _ := fsys.(*filesys); fss != nil {
			if path = path[len(fss.root):]; path != "" {
				fss.Remove(path, func(fi FileInfo) {
					mltyfsys.Notify(fss, fi, NOTE_REMOVE)
				})
			}
		}
	}
}

func (mltyfsys *multifilesys) EventCreate(path string, fsys FileSystem) {
	if fsys != nil {
		if fss, _ := fsys.(*filesys); fss != nil {
			if path = path[len(fss.root):]; path != "" {
				fss.AutoSync(path, func(fi FileInfo, ntfy Notify) {
					mltyfsys.Notify(fss, fi, NOTE_CREATE)
					mltyfsys.Notify(fss, fi, NOTE_AMMEND)
				})
			}
		}
	}
}

func (mltyfsys *multifilesys) EventWrite(path string, fsys FileSystem) {
	if fsys != nil {
		if fss, _ := fsys.(*filesys); fss != nil {
			if path = path[len(fss.root):]; path != "" {
				fss.AutoSync(path, func(fi FileInfo, ntfy Notify) {
					mltyfsys.Notify(fss, fi, ntfy)
				})
			}
		}
	}
}

func (mltyfsys *multifilesys) Open(path string) (fl File) {
	return mltyfsys.OpenContext(context.Background(), path)
}

func findFsys(ctx context.Context, mltyfsys *multifilesys, path string) (systmsfnd []FileSystem, rmngpaths []string) {
	if fsystms := mltyfsys.fsystms; fsystms != nil {
		var fsys FileSystem
		pthl := len(path)
		prfx := func() string {
			if path[0] == '/' {
				return ""
			}
			return "/"
		}()
		for pri := range pthl {
			if path[pri] == '/' {
				if fsys = fsystms[prfx+path[:pri+1]]; fsys != nil {
					if ctx != nil {
						if ctx.Err() != nil {
							return
						}
					}
					systmsfnd = append(systmsfnd, fsys)
					rmngpaths = append(rmngpaths, path[pri:])
					continue
				}
			}
			if path[pthl-(pri+1)] == '/' {
				if fsys = fsystms[prfx+path[:pthl-(pri+1)]]; fsys != nil {
					systmsfnd = append(systmsfnd, fsys)
					rmngpaths = append(rmngpaths, path[pthl-(pri):])
					continue
				}
			}
		}
	}
	return
}

func (mltyfsys *multifilesys) OpenContext(ctx context.Context, path string) (fl File) {
	if mltyfsys == nil {
		return
	}
	if fsystms := mltyfsys.fsystms; fsystms != nil {
		var fsys FileSystem
		pthl := len(path)
		prfx := func() string {
			if path[0] == '/' {
				return ""
			}
			return "/"
		}()
		for pri := range pthl {
			if path[pri] == '/' {
				if fsys = fsystms[prfx+path[:pri+1]]; fsys != nil {
					if ctx != nil {
						if ctx.Err() != nil {
							return
						}
					}
					if fl = fsys.OpenContext(ctx, path[pri:]); fl != nil {
						return fl
					}
				}
			}
			if path[pthl-(pri+1)] == '/' {
				if fsys = fsystms[prfx+path[:pthl-(pri+1)]]; fsys != nil {
					if fl = fsys.OpenContext(ctx, path[pthl-(pri+1):]); fl != nil {
						return fl
					}
					continue
				}
			}
		}
	}
	return
}

type contextreader struct {
	funcrd func([]byte) (int, error)
	cls    io.Closer
	sk     io.Seeker
	bfrdr  *bufio.Reader
}

func (ctxrdr *contextreader) Seek(offset int64, whence int) (n int64, err error) {
	if ctxrdr == nil {
		return
	}
	if sk := ctxrdr.sk; sk != nil {
		if n, err = sk.Seek(offset, whence); err == nil {
			if bfrdr := ctxrdr.bfrdr; bfrdr != nil {
				bfrdr.Reset(ctxrdr)
			}
		}
	}
	return
}

func (ctxrdr *contextreader) Close() (err error) {
	if ctxrdr == nil {
		return
	}
	ctxrdr.funcrd = nil
	ctxrdr.sk = nil
	cls := ctxrdr.cls
	ctxrdr.cls = nil
	if cls != nil {
		cls.Close()
	}
	return
}

func (ctxrdr *contextreader) ReadRune() (r rune, size int, err error) {
	if ctxrdr == nil {
		return 0, 0, io.EOF
	}
	bfrdr := ctxrdr.bfrdr
	if bfrdr != nil {
		r, size, err = bfrdr.ReadRune()
		return
	}
	if funcrdr := ctxrdr.funcrd; funcrdr != nil {
		bfrdr = bufio.NewReader(iorw.ReadFunc(func(p []byte) (n int, err error) {
			return funcrdr(p)
		}))
		ctxrdr.bfrdr = bfrdr
		r, size, err = bfrdr.ReadRune()
	}
	return
}

func (ctxrdr *contextreader) Read(b []byte) (n int, err error) {
	if ctxrdr == nil {
		return 0, io.EOF
	}
	funcrdr, bfrdr := ctxrdr.funcrd, ctxrdr.bfrdr
	if bfrdr != nil {
		n, err = bfrdr.Read(b)
		if err != nil || n == 0 {
			ctxrdr.bfrdr = nil
		}
		return
	}
	if funcrdr == nil {
		return 0, io.EOF
	}
	n, err = funcrdr(b)
	return
}

func ContextReader(rdr io.Reader, ctx context.Context) (ctxrdr *contextreader) {
	ctxrdr = &contextreader{}
	ctxrdr.cls, _ = rdr.(io.Closer)
	ctxrdr.sk, _ = rdr.(io.Seeker)
	if rdr != nil {
		if ctx != nil {
			ctxrdr.funcrd = func(b []byte) (n int, err error) {
				if err = ctx.Err(); err != nil {
					if err == context.Canceled {
						err = io.EOF
					}
					return
				}
				n, err = rdr.Read(b)
				return
			}
		} else {
			ctxrdr.funcrd = func(b []byte) (n int, err error) {
				n, err = rdr.Read(b)
				return
			}
		}
	}
	return
}
