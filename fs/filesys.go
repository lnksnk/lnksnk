package fs

import (
	"bufio"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lnksnk/lnksnk/iorw"
	"github.com/lnksnk/lnksnk/mimes"
)

type FileSystem interface {
	Path() string
	AutoSync(string, ...func(FileInfo, Notify))
	CacheExtensions(...string)
	ActiveExtensions(...string)
	DefaultExtensions(...string)
	Open(string) File
	OpenContext(context.Context, string) File
	List(string) []FileInfo
	Exist(string) bool
	Cached(string) bool
	Remove(string, ...func(FileInfo)) bool
	Set(string, ...interface{}) bool
	Touch(string) bool
	Close() error
	Stat(string) FileInfo
	StatContext(context.Context, string) FileInfo
}

type filesys struct {
	mltypath string
	root     string
	embed    map[string]*embedfile
	cachexts map[string]bool
	//cachedfiles *sync.Map
	cachedfiles map[string]*cachedstat
	defaultexts map[string]bool
	activexts   map[string]bool
	mltyfsys    MultiFileSystem
}

// Path implements FileSystem.
func (fsys *filesys) Path() string {
	if fsys == nil {
		return ""
	}
	return fsys.mltypath
}

// Remove implements FileSystem.
func (fsys *filesys) Remove(path string, rmvfunc ...func(FileInfo)) (diddel bool) {
	if fsys == nil || path == "" {
		return false
	}
	ext := filepath.Ext(path)
	cachedfiles, cachexts, embed := fsys.cachedfiles, fsys.cachexts, fsys.embed
	if ext != "" && cachexts[ext] {
		ebd := func() *embedfile {
			if embed != nil {
				return embed[path]
			}
			return nil
		}()
		chdsts := func() *cachedstat {
			if cachedfiles != nil {
				return cachedfiles[path]
			}
			return nil
		}()
		if ebd != nil {
			delete(fsys.embed, path)
			if chdsts != nil {
				delete(cachedfiles, path)
				chdsts.Close()
			}
			if len(rmvfunc) > 0 && rmvfunc[0] != nil {
				rmvfunc[0](ebd.FileInfo)
			}
			ebd.Close()
			if !diddel {
				diddel = true
			}
		}
		if chdsts != nil {
			delete(cachedfiles, path)
			if len(rmvfunc) > 0 && rmvfunc[0] != nil {
				rmvfunc[0](chdsts.FileInfo)
			}
			chdsts.Close()
			if !diddel {
				diddel = true
			}
		}
		return diddel
	}
	if ext == "" && path[0] == '/' {
		pthl := len(path)
		if path[pthl-1] != '/' {
			pthl++
			path += "/"
		}

		for epth, ebd := range embed {
			if epthl := len(epth); epthl > pthl && epth[:pthl] == path {
				if !diddel {
					diddel = true
				}
				if cachedfiles != nil {
					if chdsts := cachedfiles[epth]; chdsts != nil {
						delete(cachedfiles, epth)
						chdsts.Close()
					}
				}
				delete(embed, epth)
				if len(rmvfunc) > 0 && rmvfunc[0] != nil {
					rmvfunc[0](ebd.FileInfo)
				}
				ebd.Close()
			}
		}

		for chdpth, chdsts := range embed {
			if chdpthl := len(chdpth); chdpthl > pthl && chdpth[:pthl] == path {
				if !diddel {
					diddel = true
				}
				if embed != nil {
					if emd := embed[chdpth]; emd != nil {
						delete(embed, chdpth)
						emd.Close()
					}
				}
				delete(cachedfiles, chdpth)
				if len(rmvfunc) > 0 && rmvfunc[0] != nil {
					rmvfunc[0](chdsts.FileInfo)
				}
				chdsts.Close()
			}
		}
	}
	return diddel
}

type embedfile struct {
	*iorw.Buffer
	FileInfo
}

// Set implements FileSystem.
func (fsys *filesys) Set(path string, a ...interface{}) bool {
	if fsys == nil || path == "" {
		return false
	}
	ext := filepath.Ext(path)
	if ext == "" {
		return false
	}
	embed := fsys.embed
	if path[0] != '/' {
		path = "/" + path
	}
	if embed == nil {
		fsys.embed = map[string]*embedfile{}
		embed = fsys.embed
	}
	if emd, ok := embed[path]; ok {
		emd.Clear()
		emd.Print(a...)
		fi, _ := emd.FileInfo.(*fileinfo)
		if fi != nil {
			fi.modfy = time.Now()
			fi.size = emd.Buffer.Size()
			if cachedfiles := fsys.cachedfiles; cachedfiles != nil && fsys.activexts[ext] {
				chdst := cachedfiles[path]
				if chdst == nil {
					chdst = &cachedstat{Buffer: emd.Buffer, FileInfo: emd.FileInfo}
					cachedfiles[path] = chdst
				} else {
					chdst.Buffer = emd.Buffer
					chdst.FileInfo = emd.FileInfo
				}
			}
		}

		if fsys.mltyfsys != nil {
			fsys.mltyfsys.Notify(fsys, emd.FileInfo, NOTE_AMMEND)
		}
		return true
	}
	_, _, media := mimes.FindMimeType(ext)
	name := func() string {
		if sep := strings.LastIndex(path, "/"); sep > -1 {
			return path[sep+1:]
		}
		return path
	}()
	emd := &embedfile{Buffer: iorw.NewBuffer(a...)}
	emd.FileInfo = NewFileInfo(name, emd.Buffer.Size(), 0, time.Now(), false, nil, fsys.activexts[ext], !fsys.cachexts[ext], media, path, fsys.mltypath, func(ctx ...context.Context) io.Reader {
		if len(ctx) == 1 && ctx[0] != nil {
			return emd.Buffer.Reader(ctx)
		}
		return emd.Buffer.Reader()
	})
	embed[path] = emd
	if cachedfiles := fsys.cachedfiles; cachedfiles != nil && fsys.activexts[ext] {
		chdst := cachedfiles[path]
		if chdst == nil {
			chdst = &cachedstat{Buffer: emd.Buffer, FileInfo: emd.FileInfo}
			cachedfiles[path] = chdst
		} else {
			chdst.Buffer = emd.Buffer
			chdst.FileInfo = emd.FileInfo
		}
	}
	if fsys.mltyfsys != nil {
		fsys.mltyfsys.Notify(fsys, emd.FileInfo, NOTE_CREATE)
	}
	return true
}

// Touch implements FileSystem.
func (fsys *filesys) Touch(path string) bool {
	if fsys == nil || path == "" {
		return false
	}
	return false
}

// AttachMultiFileSystem implements FileSystem.
func (fsys *filesys) AttachMultiFileSystem(mltifsys MultiFileSystem) {
	if fsys == nil || mltifsys == nil {
		return
	}
	if fsys.mltyfsys != mltifsys {
		fsys.mltyfsys = mltifsys
	}
}

// Cached implements FileSystem.
func (fsys *filesys) Cached(path string) bool {
	if fsys == nil {
		return false
	}
	if cachedfiles := fsys.cachedfiles; cachedfiles != nil {
		return cachedfiles[path] != nil

	}
	return false
}

// Exist implements FileSystem.
func (fsys *filesys) Exist(path string) bool {
	if fsys.Cached(path) {
		return true
	}
	if fsys != nil {
		if emded := fsys.embed; emded != nil {
			_, ek := emded[path]
			if ek {
				return ek
			}
		}
		if fi, _ := os.Stat(fsys.root + path); fi != nil {
			return true
		}
	}
	return false
}

// List implements FileSystem.
func (fsys *filesys) List(path string) (fis []FileInfo) {
	if fsys == nil {
		return
	}
	pthl := len(path)

	pn := 0
	for pn < pthl {
		if path[pn] == '/' {
		retry:
			if pn < pthl-4 && path[pn:pn+4] == "/../" {
				path = path[:pn+1] + path[pn+4:]
				pthl -= 4
				goto retry
			}
		}
		pn++
	}
	name := ""
	lstspi := strings.LastIndex(path, "/")

	if lstspi == -1 {
		name = path
		path = "/"
	} else if lstspi > -1 && path != "" {
		name = path[lstspi+1:]
		path = path[:lstspi+1]
	}
	if path != "" {
		if path[0] != '/' {
			path = "/" + path
		}
		if path[len(path)-1] != '/' {
			path = path + "/"
		}
	}
	fldfis := map[string]bool{}
	var isvalid = func(chkpth string, chkmask string, firoot, fipath, finame string) (valid bool) {
		if !fldfis[fipath] {
			if firtl := len(firoot); firtl >= len(chkpth) && firoot[len(chkpth)-1] == '/' {
				if firoot == chkpth {
					if valid = chkmask == ""; valid {
						fldfis[fipath] = true
						return
					}

					if valid = strings.ContainsAny(chkmask, "*.?"); valid {
						if valid = strings.ContainsAny(chkmask, "*?"); valid {
							if valid = checkPathMask(finame, chkmask); valid {
								fldfis[fipath] = true
								return
							}
							if valid = checkPathMask(finame, chkmask); valid {
								fldfis[fipath] = true
								return
							}
						}
						if valid = strings.Contains(chkmask, "."); valid {
							if valid = checkPathMask(finame, chkmask); valid {
								fldfis[fipath] = true
								return
							}
						}
						return
					}
					return
				}
			}
		}
		return
	}
	var findosfinfos = func(path string, mask string) (osfis []FileInfo) {
		osdirs, _ := os.ReadDir(fsys.root + path)
		for _, osdir := range osdirs {
			if !osdir.IsDir() {
				osfi, _ := osdir.Info()
				if osfi != nil {
					fipath := fsys.mltypath + path + osfi.Name()
					fipthroot := fipath[:strings.LastIndex(fipath, "/")+1]
					if fipthroot != "/" && fipthroot[0] != '/' {
						fipthroot = "/" + fipthroot
					}
					if vld := isvalid(fsys.mltypath, mask, fipthroot, osfi.Name(), fipath); vld {
						_, _, media := mimes.FindMimeType(osfi.Name())
						osfis = append(osfis, &fsysfinfo{ctx: nil, FileInfo: NewFileInfo(osfi.Name(), osfi.Size(), osfi.Mode(), osfi.ModTime(), false, osfi.Sys(), fsys.activexts[filepath.Ext(osfi.Name())], true, media, fipath, fsys.mltypath, func(ctx ...context.Context) io.Reader {
							if f, _ := os.Open(fsys.root + fipath); f != nil {
								return ContextReader(f, func() context.Context {
									if len(ctx) > 0 {
										return ctx[0]
									}
									return nil
								}())
							}
							return nil
						})})
					}
				}
			}
		}
		return
	}
	var names []string
	if name == "" {
		names = append(names, name)
	} else {
		names = append(names, strings.Split(name, ",")...)
	}
	chknmdups := map[string]bool{}
	nmsi := 0
	nmsl := len(names)
	for nmsi < nmsl {
		names[nmsi] = strings.TrimFunc(names[nmsi], iorw.IsSpace)
		if chknmdups[names[nmsi]] {
			names = append(names[:nmsi], names[nmsi+1:]...)
			nmsl--
			continue
		}
		chknmdups[names[nmsi]] = true
		nmsi++
	}
	for _, name := range names {
		if path == "/" {
			for _, emd := range fsys.embed {
				vldfi := emd.FileInfo
				if vld := isvalid(fsys.mltypath, name, vldfi.Root(), vldfi.Name(), vldfi.Path()); vld {
					fis = append(fis, vldfi)
				}
			}
			for _, chdf := range fsys.cachedfiles {
				vldfi := chdf.FileInfo
				if vld := isvalid(fsys.mltypath, name, vldfi.Root(), vldfi.Name(), vldfi.Path()); vld {
					fis = append(fis, vldfi)
				}
			}
			fis = append(fis, findosfinfos("", name)...)
			continue
		}
		for _, emd := range fsys.embed {
			vldfi := emd.FileInfo
			if vld := isvalid(fsys.mltypath+path[1:], name, vldfi.Root(), vldfi.Name(), vldfi.Path()); vld {
				fis = append(fis, vldfi)
			}
		}
		for _, chdf := range fsys.cachedfiles {
			vldfi := chdf.FileInfo
			if vld := isvalid(fsys.mltypath+path[1:], name, vldfi.Root(), vldfi.Name(), vldfi.Path()); vld {
				fis = append(fis, vldfi)
			}
		}
		fis = append(fis, findosfinfos(path, name)...)
		continue
	}
	return
}

func checkPathMask(path string, mask string) (vld bool) {
	vld, _ = filepath.Match(mask, path)
	return
}

func syncPath(fsys *filesys, path string, nftyfunc func(FileInfo, Notify)) {
	if fsys == nil {
		return
	}
	var syncthis func(sncpath string)
	cachexts, cachedfiles := fsys.cachexts, fsys.cachedfiles
	syncthis = func(sncpath string) {
		if sncpath == "" {
			return
		}
		if sncpath[0] != '/' {
			sncpath = "/" + sncpath
		}

		var sncfi, _ = os.Stat(fsys.root + sncpath)

		if sncfi != nil {
			if sncfi.IsDir() {
				if sncpath[len(sncpath)-1] != '/' {
					sncpath += "/"
				}
				if sncpath != "/" && nftyfunc != nil {
					if mltyfs, _ := fsys.mltyfsys.(*multifilesys); mltyfs != nil {
						mltyfs.AutoSync(fsys.root+sncpath[:len(sncpath)-1], fsys)
					}
				}
				dirs, _ := os.ReadDir(fsys.root + sncpath)
				for _, dre := range dirs {
					drename := dre.Name()
					if dre.IsDir() {
						syncthis(sncpath + drename + "/")
						continue
					}
					syncthis(sncpath + drename)
				}
				return
			}
			if sncext := filepath.Ext(sncpath); sncext != "" && cachexts[sncext] {
				chdst, chdstok := cachedfiles[sncpath]
				if chdst == nil {
					goto loadchdsts
				}
				if chdst.ModTime() == sncfi.ModTime() {
					return
				}
				if nftyfunc != nil {
					if sncfi.Size() == 0 {
						nftyfunc(chdst.FileInfo, NOTE_REMOVE)
					}
				}
				chdst.Close()
				delete(cachedfiles, sncpath)
			loadchdsts:
				if sncfi.Size() > 0 {
					if f, _ := os.Open(fsys.root + sncpath); f != nil {
						defer f.Close()
						bf := iorw.NewBuffer(f)
						atv := fsys.activexts[sncext]
						_, _, media := mimes.FindMimeType(sncext, sncext)
						chdst = &cachedstat{Buffer: bf, FileInfo: NewFileInfo(sncfi.Name(), sncfi.Size(), sncfi.Mode(), sncfi.ModTime(), sncfi.IsDir(), sncfi.Sys(), atv, !atv, media, sncpath, fsys.mltypath, func(ctx ...context.Context) io.Reader { return bf.Reader(ctx) })}
						cachedfiles[sncpath] = chdst
						if nftyfunc != nil {
							if chdstok {
								nftyfunc(chdst.FileInfo, NOTE_AMMEND)
							} else {
								nftyfunc(chdst.FileInfo, NOTE_CREATE)
							}
						}
					}
				}
			}
		}
	}
	syncthis(path)
}

func (fsys *filesys) AutoSync(path string, ntfyfunc ...func(FileInfo, Notify)) {
	if fsys == nil {
		return
	}
	syncPath(fsys, path, func() func(FileInfo, Notify) {
		if len(ntfyfunc) > 0 {
			return ntfyfunc[0]
		}
		return nil
	}())
}

func (fsys *filesys) Close() (err error) {
	if fsys == nil {
		return
	}
	mltyfsys, activexts, cachexts, defaultexts, cachedfiles := fsys.mltyfsys, fsys.activexts, fsys.cachexts, fsys.defaultexts, fsys.cachedfiles
	fsys.mltyfsys = nil
	if mltsys, _ := mltyfsys.(*multifilesys); mltsys != nil {
		if root := fsys.root; root != "" {
			if fsystms := mltsys.fsystms; fsystms != nil {
				fsystms.Delete(fsys.mltypath)
			}
		}
	}
	fsys.defaultexts = nil
	fsys.cachexts = nil
	fsys.activexts = nil
	if activexts != nil {
		clear(activexts)
		activexts = nil
	}
	if defaultexts != nil {
		clear(defaultexts)
		defaultexts = nil
	}
	if cachexts != nil {
		clear(cachexts)
		cachexts = nil
	}
	for key, value := range cachedfiles {
		value.Close()
		delete(cachedfiles, key)
	}
	return
}

func NewFileSystem(root string) FileSystem {
	return &filesys{root: strings.Replace(root, "\\", "/", -1), cachedfiles: map[string]*cachedstat{}, cachexts: map[string]bool{}, activexts: map[string]bool{}, defaultexts: map[string]bool{}}
}

func (fsys *filesys) CacheExtensions(extns ...string) {
	if fsys == nil {
		return
	}
	chdexts := fsys.cachexts
	if chdexts == nil {
		fsys.cachexts = map[string]bool{}
		chdexts = fsys.cachexts
	}
	for _, ext := range extns {
		if !chdexts[ext] {
			chdexts[ext] = true
		}
	}
}

func (fsys *filesys) ActiveExtensions(extns ...string) {
	if fsys == nil {
		return
	}
	activexts := fsys.activexts
	if activexts == nil {
		fsys.activexts = map[string]bool{}
		activexts = fsys.activexts
	}
	for _, ext := range extns {
		if !activexts[ext] {
			activexts[ext] = true
		}
	}
}

func (fsys *filesys) DefaultExtensions(extns ...string) {
	if fsys == nil {
		return
	}
	defaultexts := fsys.defaultexts
	if defaultexts == nil {
		fsys.defaultexts = map[string]bool{}
		defaultexts = fsys.defaultexts
	}
	for _, ext := range extns {
		if !defaultexts[ext] {
			defaultexts[ext] = true
		}
	}
}

func (fsys *filesys) Open(path string) File {
	return fsys.OpenContext(context.Background(), path)
}

type fsysfinfo struct {
	FileInfo
	reader io.Reader
	ctx    context.Context
}

func (fsysfi *fsysfinfo) Close() (err error) {
	if fsysfi == nil {
		return
	}
	fsysfi.FileInfo = nil
	reader := fsysfi.reader
	fsysfi.reader = nil
	fsysfi.ctx = nil
	if reader != nil {
		if cls := reader.(io.Closer); cls != nil {
			cls.Close()
		}
	}
	return
}

func (fsysfi *fsysfinfo) Reader(ctx ...context.Context) (reader io.Reader) {
	if fsysfi == nil {
		return nil
	}
	if reader = fsysfi.reader; reader == nil {
		fsysfi.reader = fsysfi.FileInfo.Reader(ctx...)
		return fsysfi.reader
	}
	return fsysfi.reader
}

// Stat implements FileSystem.
func (fsys *filesys) Stat(path string) FileInfo {
	return fsys.StatContext(context.Background(), path)
}

// StatContext implements FileSystem.
func (fsys *filesys) StatContext(ctx context.Context, path string) (fifnd FileInfo) {
	if fsys == nil {
		return
	}
	pthl := len(path)

	pn := 0
	for pn < pthl {
		if path[pn] == '/' {
		retry:
			if pn < pthl-4 && path[pn:pn+4] == "/../" {
				path = path[:pn+1] + path[pn+4:]
				pthl -= 4
				goto retry
			}
		}
		pn++
	}
	pthext := filepath.Ext(path)
	pthroot := func() (rt string) {
		if pthext == "" {
			if rt = path; (rt != "" && rt[len(rt)-1] != '/') || rt == "" {
				rt += "/"
				path = ""
				return
			}
			path = ""
			return
		}
		if lsti := strings.LastIndex(path, "/"); lsti > -1 {
			rt = path[:lsti+1]
			path = path[lsti+1:]
			return
		}
		return "/"
	}()
	if pthroot != "" && pthroot[0] != '/' {
		pthroot = "/" + pthroot
	}
	if path == "" && pthroot[len(pthroot)-1] == '/' {
		for dlftext := range fsys.defaultexts {
			if emdfi := fsys.embed[pthroot+"index"+dlftext]; emdfi != nil {
				if fsys.cachexts[emdfi.FileInfo.Ext()] {
					var chdstt *cachedstat = fsys.cachedfiles[pthroot+path]
					if chdstt != nil {
						return chdstt.FileInfo
					}
					go func() {
						fsys.cachedfiles[pthroot+path] = &cachedstat{Buffer: emdfi.Buffer, FileInfo: emdfi.FileInfo}
					}()
				}
				return emdfi.FileInfo
			}
		}
	}
	if emdfi := fsys.embed[pthroot+path]; emdfi != nil {
		if fsys.cachexts[emdfi.FileInfo.Ext()] {
			var chdstt *cachedstat = fsys.cachedfiles[pthroot+path]
			if chdstt != nil {
				return chdstt.FileInfo
			}
			go func() {
				fsys.cachedfiles[pthroot+path] = &cachedstat{Buffer: emdfi.Buffer, FileInfo: emdfi.FileInfo}
			}()
		}
		return emdfi.FileInfo
	}
	var fi fs.FileInfo
	if path == "" && pthroot[len(pthroot)-1] == '/' {
		fi, _ := os.Stat(fsys.root + pthroot + path)
		if fi != nil && fi.IsDir() {
			for dlftext := range fsys.defaultexts {
				if fi, _ = os.Stat(fsys.root + pthroot + "index" + dlftext); fi != nil {
					path = fi.Name()
					pthext = filepath.Ext(path)
					break
				}
			}
			if fi == nil {
				return nil
			}
		}
	}
	actv := fsys.activexts[pthext]
	media := false
	raw := !actv

	if chble := func() bool {
		return pthext != "" && fsys.cachexts != nil && fsys.cachexts[pthext]
	}(); chble {
		var chdstt *cachedstat = fsys.cachedfiles[pthroot+path]
		if chdstt != nil {
			return chdstt.FileInfo
		}
		/*go func() {
			if f, _ := os.Open(fsys.root + pthroot + path); f != nil {
				bf := iorw.NewBuffer(f)
				chdstt = &cachedstat{Buffer: bf, FileInfo: NewFileInfo(fi.Name(), fi.Size(), fi.Mode(), fi.ModTime(), fi.IsDir(), fi.Sys(), !fi.IsDir() && fsys.activexts[filepath.Ext(fi.Name())], raw, media, func() string {
					if pthroot != "" && pthroot[0] == '/' {
						return pthroot[1:]
					}
					return pthroot
				}()+path, fsys.mltypath, func(ctx ...context.Context) io.Reader {
					return bf.Reader(func() context.Context {
						if len(ctx) > 0 {
							return ctx[0]
						}
						return nil
					}())
				})}
				fsys.cachedfiles[pthroot+path] = chdstt
			}
		}()*/
	}

	fi, _ = os.Stat(fsys.root + pthroot + path)
	if fi != nil && !fi.IsDir() {
		_, _, media = mimes.FindMimeType(fi.Name())
		fifnd = &fsysfinfo{ctx: ctx, FileInfo: NewFileInfo(fi.Name(), fi.Size(), fi.Mode(), fi.ModTime(), false, fi.Sys(), fsys.activexts[filepath.Ext(fi.Name())], raw, media, func() string {
			if pthroot != "" && pthroot[0] == '/' {
				return pthroot[1:]
			}
			return pthroot
		}()+path, fsys.mltypath, func(ctx ...context.Context) io.Reader {
			if f, _ := os.Open(fsys.root + pthroot + path); f != nil {
				return ContextReader(f, func() context.Context {
					if len(ctx) > 0 {
						return ctx[0]
					}
					return nil
				}())
			}
			return nil
		})}

	}
	return
}

func (fsys *filesys) OpenContext(ctx context.Context, path string) File {
	if ctx == nil {
		ctx = context.Background()
	}
	if fsys == nil {
		return nil
	}
	pthext := filepath.Ext(path)
	pthroot := func() (rt string) {
		if pthext == "" {
			if rt = path; (rt != "" && rt[len(rt)-1] != '/') || rt == "" {
				rt += "/"
				path = ""
				return
			}
			path = ""
			return
		}
		if lsti := strings.LastIndex(path, "/"); lsti > -1 {
			rt = path[:lsti+1]
			path = path[lsti+1:]
			return
		}
		return "/"
	}()
	if pthroot != "" && pthroot[0] != '/' {
		pthroot = "/" + pthroot
	}
	var fi fs.FileInfo
	if path == "" && pthroot[len(pthroot)-1] == '/' {
		fi, _ := os.Stat(fsys.root + pthroot + path)
		if fi != nil && fi.IsDir() {
			for dlftext := range fsys.defaultexts {
				if fi, _ = os.Stat(fsys.root + pthroot + "index" + dlftext); fi != nil {
					path = fi.Name()
					pthext = filepath.Ext(path)
					break
				}
			}
			if fi == nil {
				return nil
			}
		}
	}
	actv := fsys.activexts[pthext]
	media := false
	raw := !actv

	if chble := func() bool {
		return pthext != "" && fsys.cachexts != nil && fsys.cachexts[pthext]
	}(); chble {
		var chdstt *cachedstat = fsys.cachedfiles[pthroot+path]
		if chdstt != nil {
			return &file{ctx: ctx, chd: true, fullpath: fsys.root + pthroot + path, FileInfo: chdstt.FileInfo, mxread: -1}
		}
		go func() {
			if f, _ := os.Open(fsys.root + pthroot + path); f != nil {
				bf := iorw.NewBuffer(f)
				chdstt = &cachedstat{Buffer: iorw.NewBuffer(f), FileInfo: NewFileInfo(fi.Name(), fi.Size(), fi.Mode(), fi.ModTime(), fi.IsDir(), fi.Sys(), !fi.IsDir() && fsys.activexts[filepath.Ext(fi.Name())], raw, media, func() string {
					if pthroot != "" && pthroot[0] == '/' {
						return pthroot[1:]
					}
					return pthroot
				}()+path, fsys.mltypath, func(ctx ...context.Context) io.Reader {
					return bf.Reader(func() context.Context {
						if len(ctx) > 0 {
							return ctx[0]
						}
						return nil
					}())
				})}
				fsys.cachedfiles[pthroot+path] = chdstt
			}
		}()
	}

	fi, _ = os.Stat(fsys.root + pthroot + path)
	if fi != nil && !fi.IsDir() {
		_, _, media = mimes.FindMimeType(fi.Name())
		return &file{ctx: ctx, fullpath: fsys.root + pthroot + path, FileInfo: NewFileInfo(fi.Name(), fi.Size(), fi.Mode(), fi.ModTime(), fi.IsDir(), fi.Sys(), !fi.IsDir() && fsys.activexts[filepath.Ext(fi.Name())], raw, media, func() string {
			if pthroot != "" && pthroot[0] == '/' {
				return pthroot[1:]
			}
			return pthroot
		}()+path, fsys.mltypath, func(ctx ...context.Context) io.Reader {
			if f, _ := os.Open(fsys.root + pthroot + path); f != nil {
				return ContextReader(f, func() context.Context {
					if len(ctx) > 0 {
						return ctx[0]
					}
					return nil
				}())
			}
			return nil
		}), mxread: -1}
	}
	return nil
}

type fileinfo struct {
	name   string
	path   string
	root   string
	base   string
	ext    string
	size   int64
	mode   fs.FileMode
	modfy  time.Time
	dir    bool
	active bool
	raw    bool
	media  bool
	sys    interface{}
	rdr    func(...context.Context) io.Reader
}

func NewFileInfo(name string, size int64, mode fs.FileMode, modfy time.Time, dir bool, sys any, active bool, raw bool, media bool, path, base string, rdrfunc func(...context.Context) io.Reader) (fi *fileinfo) {
	var root = func() string {
		if sepi := strings.LastIndex(path, "/"); sepi > -1 {
			return path[:sepi+1]
		}
		return "/"
	}()
	if base != "" {
		if path == "" {
			path = base
		} else {
			if path[0] == '/' {
				if base[len(base)-1] == '/' {
					path = base + path[1:]
				} else {
					path = base + path
				}
			} else {
				if base[len(base)-1] == '/' {
					path = base + path
				} else {
					path = base + "/" + path
				}
			}
		}
		if root[0] == '/' {
			if base[len(base)-1] == '/' {
				root = base + root[1:]
			} else {
				root = base + root
			}
		} else {
			if base[len(base)-1] == '/' {
				root = base + root
			} else {
				root = base + "/" + root
			}
		}
	}
	fi = &fileinfo{name: name, size: size, mode: mode, modfy: modfy, dir: dir, sys: sys, active: active, raw: raw, media: media, path: path, root: root, base: base, rdr: rdrfunc}
	if !dir {
		fi.ext = filepath.Ext(name)
	}
	return
}

func (finfo *fileinfo) Reader(ctx ...context.Context) io.Reader {
	if finfo == nil {
		return nil
	}
	if rdr := finfo.rdr; rdr != nil {
		return rdr(ctx...)
	}
	return nil
}

func (finfo *fileinfo) Path() string {
	if finfo == nil {
		return ""
	}
	return finfo.path
}

func (finfo *fileinfo) Root() string {
	if finfo == nil {
		return ""
	}
	return finfo.root
}

func (finfo *fileinfo) Base() string {
	if finfo == nil {
		return ""
	}
	return finfo.base
}

func (finfo *fileinfo) Name() string {
	if finfo == nil {
		return ""
	}
	return finfo.name
}

func (finfo *fileinfo) Size() int64 {
	if finfo == nil {
		return 0
	}
	return finfo.size
}

func (finfo *fileinfo) Mode() fs.FileMode {
	if finfo == nil {
		return 0
	}
	return finfo.mode
}

func (finfo *fileinfo) ModTime() time.Time {
	if finfo == nil {
		return time.Now()
	}
	return finfo.modfy
}

func (finfo *fileinfo) IsDir() bool {
	if finfo == nil {
		return false
	}
	return finfo.dir
}

func (finfo *fileinfo) Sys() any {
	if finfo == nil {
		return nil
	}
	return finfo.sys
}

func (finfo *fileinfo) Active() bool {
	if finfo == nil {
		return false
	}
	return finfo.active
}

func (finfo *fileinfo) Raw() bool {
	if finfo == nil {
		return false
	}
	return finfo.raw
}

func (finfo *fileinfo) Ext() (ext string) {
	if finfo == nil {
		return ext
	}
	ext = finfo.ext
	return
}

func (finfo *fileinfo) Media() bool {
	if finfo == nil {
		return false
	}
	return finfo.media
}

type file struct {
	ctx context.Context
	FileInfo
	chd      bool
	fullpath string
	bfr      *bufio.Reader
	r        io.ReadSeekCloser
	mxread   int64
}

// Active implements File.
func (f *file) Active() bool {
	if f == nil || f.FileInfo == nil {
		return false
	}
	return f.FileInfo.Active()
}

// Media implements File.
func (f *file) Media() bool {
	if f == nil || f.FileInfo == nil {
		return false
	}
	return f.FileInfo.Media()
}

// Raw implements File.
func (f *file) Raw() bool {
	if f == nil || f.FileInfo == nil {
		return false
	}
	return f.FileInfo.Raw()
}

// Close implements http.File.
func (f *file) Close() (err error) {
	if f == nil {
		return nil
	}
	r := f.r
	f.r = nil
	if r != nil {
		r.Close()
	}
	f.FileInfo = nil
	f.bfr = nil

	return
}

func (f *file) SetMaxRead(maxread int64) {
	if f == nil {
		return
	}
	f.mxread = maxread
}

// Read implements http.File.
func (f *file) Read(p []byte) (n int, err error) {
	if f == nil {
		return 0, io.EOF
	}
	if f.mxread == 0 {
		return 0, io.EOF
	}
	if f.r == nil {
		if ctx := f.ctx; ctx != nil {
			if cxterr := ctx.Err(); cxterr != nil {
				err = cxterr
				return
			}
		}
		f.r = f.FileInfo.Reader(f.ctx).(io.ReadSeekCloser)
	}
	if bfr, r := f.bfr, f.r; r != nil {
		pl := len(p)
		if f.mxread > -1 && int64(pl) > f.mxread {
			pl = int(f.mxread)
		}
		if f.chd {
			if n, err = f.r.Read(p[:pl]); n > 0 && err == nil {
				if f.mxread > 0 {
					f.mxread -= int64(n)
				}
			}
			return
		}
		if bfr == nil {
			f.bfr = bufio.NewReaderSize(f.r, 65787)
			bfr = f.bfr
		}
		if n, err = bfr.Read(p[:pl]); n > 0 && err == nil {
			if f.mxread > 0 {
				f.mxread -= int64(n)
			}
			return
		}
	}

	if n == 0 && err == nil {
		err = io.EOF
	}
	return
}

// Readdir implements http.File.
func (f *file) Readdir(count int) (fis []fs.FileInfo, err error) {
	if f == nil {
		return
	}
	if fullpath := f.fullpath; fullpath != "" {
		if fext := filepath.Ext(fullpath); fext == "" {
			if !strings.HasSuffix(fullpath, "/") {
				fullpath += "/"
				f.fullpath = fullpath
			}
			if fdir, _ := os.DirFS(fullpath).(fs.ReadDirFS); fdir != nil {
				fdirs, _ := fdir.ReadDir("")
				for _, fdir := range fdirs {
					if fi, _ := fdir.Info(); fi != nil {
						fis = append(fis, fi)
					}
				}
			}

		}
	}
	return
}

// Seek implements http.File.
func (f *file) Seek(offset int64, whence int) (n int64, err error) {
	if f == nil {
		return 0, nil
	}
	if sr := f.r; sr != nil {
		if n, err = sr.Seek(offset, whence); err == nil {
			if f.bfr != nil {
				f.bfr.Reset(f.r)
			}
			f.mxread = -1
		}
		return
	}
	return 0, nil
}

// Stat implements http.File.
func (f *file) Stat() FileInfo {
	if f == nil {
		return nil
	}
	return f.FileInfo
}

func (f *file) Size() int64 {
	if f == nil {
		return 0
	}
	if fi := f.Stat(); fi != nil {
		return fi.Size()
	}
	return 0
}

func (f *file) ModTime() time.Time {
	if f == nil {
		return time.Now()
	}
	if fi := f.Stat(); fi != nil {
		return fi.ModTime()
	}
	return time.Now()
}

type cachedstat struct {
	FileInfo
	*iorw.Buffer
}

func (chdstat *cachedstat) Reader(ctx ...context.Context) io.Reader {
	if chdstat == nil {
		return nil
	}
	if buffer := chdstat.Buffer; buffer != nil {
		return buffer.Reader(ctx)
	}
	return nil
}

func (chdstat *cachedstat) Close() {
	if chdstat == nil {
		return
	}
	chdstat.Buffer.Close()
	chdstat.Buffer = nil
	chdstat.FileInfo = nil
}

type FileInfo interface {
	fs.FileInfo
	Active() bool
	Raw() bool
	Media() bool
	Ext() string
	Path() string
	Root() string
	Base() string
	Reader(...context.Context) io.Reader
}

type File interface {
	FileInfo
	io.Closer
	io.Reader
	io.Seeker
	Readdir(count int) ([]fs.FileInfo, error)
	Stat() FileInfo
	Size() int64
	ModTime() time.Time
	Active() bool
	Raw() bool
	Media() bool
	SetMaxRead(int64)
}
