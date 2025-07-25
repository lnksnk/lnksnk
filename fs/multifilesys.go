package fs

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/lnksnk/lnksnk/ioext"
)

type MultiFileSystem interface {
	Load(a ...interface{})
	Unload(a ...interface{})
	Open(string) File
	OpenContext(context.Context, string) File
	List(...string) []FileInfo
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
	fsystms   FileSystems
	wtchdfsys FileSystems // map[string]FileSystem
	wtchr     *watcher
	chdexts   map[string]bool
	actvexts  map[string]bool
	dfltexts  map[string]bool
}

// Unload implements MultiFileSystem.
func (mltyfsys *multifilesys) Unload(a ...interface{}) {
	if mltyfsys == nil {
		return
	}
	var mltyfsconfig []interface{}
	fsystms, wtchdfsys, wtchr := mltyfsys.fsystms, mltyfsys.wtchdfsys, mltyfsys.wtchr
	var eventsfsysms *ioext.MapIterateEvents[string, FileSystem]
	ctxfsysms, cnclfsysms := context.WithCancel(context.Background())
	defer cnclfsysms()
	if fsystms != nil {
		eventsfsysms = fsystms.Events().(*ioext.MapIterateEvents[string, FileSystem])
	}
	var eventswtchdfys *ioext.MapIterateEvents[string, FileSystem]
	ctxwtchdfsys, cnclwtchdfsys := context.WithCancel(context.Background())
	defer cnclwtchdfsys()
	if wtchdfsys != nil {
		eventswtchdfys = wtchdfsys.Events().(*ioext.MapIterateEvents[string, FileSystem])
	}
	al := len(a)
	if al > 0 {
		var delfsys []string
		in, _ := ioext.NewBuffer(a...).Reader(true).Marshal()
		if arrs, arrk := in.([]string); arrk {
			for _, as := range arrs {
				if as != "" {
					mltyfsconfig = append(mltyfsconfig, as)
				}
			}
		} else if arri, arrik := in.([]interface{}); arrik {
			if len(arri) > 0 {
				mltyfsconfig = arri
			}
		}

		if len(mltyfsconfig) > 0 {
			for _, fsnd := range mltyfsconfig {
				if as, ask := fsnd.(string); ask && as != "" {
					delfsys = append(delfsys, as)
				}
			}
		}
		if len(delfsys) > 0 {
			if fsystms != nil {
				eventsfsysms.EventDeleted = func(dlmp map[string]FileSystem) {
					defer cnclfsysms()
					for _, dlfsys := range dlmp {
						dlfsys.Close()
					}
				}
				fsystms.Delete(delfsys...)
				<-ctxfsysms.Done()
			}
			if wtchdfsys != nil {
				eventswtchdfys.EventDeleted = func(dlmp map[string]FileSystem) {
					defer cnclwtchdfsys()
					for dlwtchpth, dlwtchfsys := range dlmp {
						if wtchr != nil {
							wtchr.Remove(dlwtchpth)
						}
						dlwtchfsys.Close()
					}
				}
				wtchdfsys.Delete(delfsys...)
				<-ctxwtchdfsys.Done()
				if wtchdfsys.Empty() {
					mltyfsys.wtchr = nil
					wtchr.Close()
				}
			}
		}
		return
	}
	if fsystms != nil {
		eventsfsysms.EventDisposed = func(dlmp map[string]FileSystem) {
			defer cnclfsysms()
			for _, dlfsys := range dlmp {
				dlfsys.Close()
			}
		}
		fsystms.Close()
		<-ctxfsysms.Done()
	}
	if wtchdfsys != nil {
		eventswtchdfys.EventDisposed = func(dlmp map[string]FileSystem) {
			defer cnclwtchdfsys()
			for dlwtchpth, dlwtchfsys := range dlmp {
				if wtchr != nil {
					wtchr.Remove(dlwtchpth)
				}
				dlwtchfsys.Close()
			}
		}
		wtchdfsys.Close()
		<-ctxwtchdfsys.Done()
		if wtchdfsys.Empty() {
			mltyfsys.wtchr = nil
			wtchr.Close()
		}
	}
}

// Load implements MultiFileSystem.
func (mltyfsys *multifilesys) Load(a ...interface{}) {
	if mltyfsys == nil || len(a) == 0 {
		return
	}
	var filesysconf map[string]interface{}
	ai := 0
	if al := len(a); al > 0 {
		for ai < al {
			if confd, confdk := a[ai].(map[string]interface{}); confdk {
				if len(confd) > 0 {
					if filesysconf == nil {
						filesysconf = confd
					} else {
						for ck, cv := range confd {
							filesysconf[ck] = cv
						}
					}
				}
				a = append(a[:ai], a[ai+1:]...)
				al--
				continue
			}
			ai++
		}
		if al > 0 {
			in, _ := ioext.NewBuffer(a...).Reader(true).Marshal()
			if cnfd, cnfk := in.(map[string]interface{}); cnfk {
				if len(cnfd) > 0 {
					if filesysconf == nil {
						filesysconf = cnfd
					} else {
						for ck, cv := range cnfd {
							filesysconf[ck] = cv
						}
					}
				}
			}
		}
	}
	if len(filesysconf) == 0 {
		return
	}
	for path, pthv := range filesysconf {
		pths, _ := pthv.(string)
		mltyfsys.Map(path, pths)
	}
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
		var fsys, _ = fsystms.Get(root)
		if fsys != nil {
			path = path[len(root):]
			return fsys.Set(path, a...)
		}
		rtl := len(root)
		for n := range rtl {
			if root[rtl-(n+1)] == '/' {
				if fsys, _ = fsystms.Get(root[:rtl-(n+1)]); fsys != nil {
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
	if path != "" && path[0] != '/' {
		path = "/" + path
	}
	if path != "" && filepath.Ext(path) == "" && path[len(path)-1] != '/' {
		path += "/"
	}
	pthl := len(path)
	pn := 0
	if pthl >= 2 && path[:2] == "./" {
		path = path[1:]
		pthl--
	}
	for pn < pthl {
		if path[pn] == '/' {
		retry:
			if fsfnd, _ := fsystms.Get(path[:func() int {
				if pn == 0 {
					return 1
				}
				return pn
			}()]); fsfnd != nil {
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
			if fsfnd, _ := fsystms.Get(path[:tpn]); fsfnd != nil {
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
			syslpthsl := len(syspaths)
			mtchdsyspths := map[string]bool{}
			if syslpthsl > 0 {
				spthsi := 0
				for spthsi < syslpthsl {
					if syspaths[spthsi] = strings.TrimFunc(syspaths[spthsi], ioext.IsSpace); syspaths[spthsi] == "" {
						syspaths = append(syspaths[:spthsi], syspaths[spthsi+1:]...)
						syslpthsl--
						continue
					}
					if mtchdsyspths[syspaths[spthsi]] {
						syspaths = append(syspaths[:spthsi], syspaths[spthsi+1:]...)
						syslpthsl--
						continue
					}
					mtchdsyspths[syspaths[spthsi]] = true
					spthsi++
				}
			}
			if syslpthsl > 0 {
				for key, value := range fsystms.Iterate() {
					if mtchdsyspths[key] {
						if !yield(key, value) {
							return
						}
					}
				}
				mtchdsyspths = nil
				return
			}
			for key, value := range fsystms.Iterate() {
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
func (mltyfsys *multifilesys) Exist(path string) bool {
	if mltyfsys == nil || path == "" {
		return false
	}

	fsystms := mltyfsys.fsystms
	pthl := len(path)
	pn := 0
	for pn < pthl {
		if path[pn] == '/' {
		retry:
			if fsfnd, _ := fsystms.Get(path[:func() int {
				if pn == 0 {
					return 1
				}
				return pn
			}()]); fsfnd != nil {
				if fsfnd.Exist(path[pn:]) {
					return true
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
			if fsfnd, _ := fsystms.Get(path[:tpn]); fsfnd != nil {
				if fsfnd.Exist(path[tpn:]) {
					return true
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
	return false
}

// List implements MultiFileSystem.
func (mltyfsys *multifilesys) List(paths ...string) (fios []FileInfo) {
	if mltyfsys == nil || len(paths) == 0 {
		return
	}

	pthsl := len(paths)
	pthsi := 0

	dppth := map[string]bool{}
	for pthsi < pthsl {
		if paths[pthsi] = strings.TrimFunc(paths[pthsi], ioext.IsSpace); paths[pthsi] == "" {
			paths = append(paths[:pthsi], paths[pthsi+1:]...)
			pthsl--
			continue
		}
		if dppth[paths[pthsi]] {
			paths = append(paths[:pthsi], paths[pthsi+1:]...)
			pthsl--
			continue
		}
		dppth[paths[pthsi]] = true
		pthsi++
	}

	alrdyfdn := map[string]bool{}
	fsystms := mltyfsys.fsystms
	for _, path := range paths {
		pthl := len(path)
		pn := 0
		for pn < pthl {
			if path[pn] == '/' {
			retry:
				if fsfnd, _ := fsystms.Get(path[:func() int {
					if pn == 0 {
						return 1
					}
					return pn
				}()]); fsfnd != nil {
					if !alrdyfdn[fsfnd.Path()] {
						alrdyfdn[fsfnd.Path()] = true
						if fis := fsfnd.List(path[pn:]); len(fis) > 0 {
							fios = append(fios, fis...)
							break
						}
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
				if fsfnd, _ := fsystms.Get(path[:tpn]); fsfnd != nil {
					if !alrdyfdn[fsfnd.Path()] {
						alrdyfdn[fsfnd.Path()] = true
						if fis := fsfnd.List(path[tpn:]); len(fis) > 0 {
							fios = append(fios, fis...)
							break
						}
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
	}
	if fiosl := len(fios); fiosl > 0 {
		fioi := 0
		chkgiosdup := map[string]bool{}
		for fioi < fiosl {
			if chkgiosdup[fios[fioi].Path()] {
				fios[fioi] = nil
				fios = append(fios[:fioi], fios[:fioi+1]...)
				fiosl--
				continue
			}
			chkgiosdup[fios[fioi].Path()] = true
			fioi++
		}
	}
	return fios
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
		raw := strings.Contains(path[0].(string), "/raw:")
		if raw {
			path[0] = strings.Replace(path[0].(string), "/raw:", "", 1)
		}
		async = strings.Contains(path[0].(string), "/sync:")
		if async {
			path[0] = strings.Replace(path[0].(string), "/sync:", "", 1)
		}
		if fsys, _ = fsystms.Get(path[0].(string)); fsys != nil {
			return
		}

		if fsys = NewFileSystem(func() string {
			if len(path) <= 1 {
				return ""
			}
			return path[1].(string)
		}(), raw); fsys != nil {
			if ntfyfsysfunc != nil && ntfyfunc == nil {
				ntfyfunc = func(fi FileInfo, n Notify) {
					ntfyfsysfunc(fsys, fi, n)
				}
			}
			if fss, _ := fsys.(*filesys); fss != nil {
				fsystms.Set(path[0].(string), fsys)
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
				if fsys.Syncable() && async && pthl == 2 {
					mltyfsys.AutoSync(path[pthl-1].(string), fsys)
					if ntfyfunc != nil {
						fsys.AutoSync("/", ntfyfunc)
					} else {
						fsys.AutoSync("/")
					}
				}
			} else {
				fsystms.Set(path[0].(string), fsys)
			}
		}
	}
	return
}

func NewMultiFileSystem() MultiFileSystem {
	return &multifilesys{ntyfiers: map[Notify]func(FileSystem, FileInfo, Notify){}, fsystms: NewFileSystems(), chdexts: map[string]bool{}, actvexts: map[string]bool{}, dfltexts: map[string]bool{}, wtchdfsys: NewFileSystems()}
}

func (mltyfsys *multifilesys) AutoSync(path string, fsys FileSystem) {
	if mltyfsys == nil && path == "" {
		return
	}

	wtchr := mltyfsys.wtchr
	trvfsys := func(pth string) (fsys FileSystem) {
		pthl := len(pth)
		if wtchdfsys := mltyfsys.wtchdfsys; wtchdfsys != nil {
			for wtchpth, wtchfsys := range wtchdfsys.Iterate() {
				if wthl := len(wtchpth); pthl >= wthl && wtchpth[:wthl] == pth[:wthl] {
					fsys = wtchfsys
					break
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
	if fsys.Syncable() {
		if wtchr.Add(path) {
			wtchdfsys.Set(path, fsys)
		}
		return
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
				if fsys, _ = fsystms.Get(prfx + path[:pri+1]); fsys != nil {
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
				if fsys, _ = fsystms.Get(prfx + path[:pthl-(pri+1)]); fsys != nil {
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
		bfrdr = bufio.NewReader(ioext.ReadFunc(func(p []byte) (n int, err error) {
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
