package resources

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lnksnk/lnksnk/fsutils"
	"github.com/lnksnk/lnksnk/iorw"
	"github.com/lnksnk/lnksnk/iorw/iocaching"
	"github.com/lnksnk/lnksnk/mimes"
	"github.com/lnksnk/lnksnk/web"
)

type EmbeddedResource struct {
	rscngendpnt *ResourcingEndpoint
	*iorw.Buffer
	modified time.Time
	path     string
}

func (embdrs *EmbeddedResource) Name() string {
	if strings.Contains(embdrs.path, "/") && strings.LastIndex(embdrs.path, "/") < strings.LastIndex(embdrs.path, ".") {
		return embdrs.path[strings.LastIndex(embdrs.path, "/")+1:]
	}
	return embdrs.path
}

func NewEmbeddedResource(rscngendpnt *ResourcingEndpoint) (embdrs *EmbeddedResource) {
	embdrs = &EmbeddedResource{Buffer: iorw.NewBuffer(), modified: time.Now(), rscngendpnt: rscngendpnt}
	return
}

func (embdrs *EmbeddedResource) fsopener(path string, a ...interface{}) (r io.ReadCloser, err error) {
	if embdrs != nil {
		if buf, rscngepnt := embdrs.Buffer, embdrs.rscngendpnt; buf != nil && rscngepnt != nil {
			r = buf.Reader()
			if len(a) > 0 {
				for _, d := range a {
					if d != nil {
						if fnrawOrActive, _ := d.(func(raw bool, active bool)); fnrawOrActive != nil {
							fnrawOrActive(rscngepnt.isRaw, rscngepnt.isActive)
						} else if fnmodified, _ := d.(func(modified time.Time)); fnmodified != nil {
							fnmodified(embdrs.modified)
						}
					}
				}
			}
		}
	}
	return
}

func (embdrs *EmbeddedResource) Clear() {
	embdrs.Buffer.Clear()
}

func (embdrs *EmbeddedResource) Close() (err error) {
	if embdrs != nil {
		if buf := embdrs.Buffer; buf != nil {
			embdrs.Buffer = nil
			err = buf.Close()
		}
		if rscngendpnt := embdrs.rscngendpnt; rscngendpnt != nil {
			embdrs.rscngendpnt = nil
			if rscngendpnt.embeddedResources[embdrs.path] == embdrs {
				rscngendpnt.embeddedResources[embdrs.path] = nil
				delete(rscngendpnt.embeddedResources, embdrs.path)
			}
		}

		embdrs = nil
	}
	return
}

// ResourcingEndpoint - struct
type ResourcingEndpoint struct {
	//lck               *sync.Mutex
	fsutils           *fsutils.FSUtils
	fs                FS
	path              string
	epnttype          string
	isLocal           bool
	isRemote          bool
	isEmbedded        bool
	isRaw             bool
	isActive          bool
	remoteHeaders     map[string]string
	host              string
	schema            string
	querystring       string
	embeddedResources map[string]*EmbeddedResource
	rsngmngr          *ResourcingManager
	cachableExtsBuffs *iocaching.BufferCache
}

// FS return fsutils.FSUtils implementation for *ResourcingEndPoint
func (rscngepnt *ResourcingEndpoint) FS() *fsutils.FSUtils {
	if rscngepnt.fsutils == nil {
		rscngepnt.fsutils = &fsutils.FSUtils{
			ABS: func(path string) (abspath string) {
				abspath, _ = rscngepnt.fsabs(path)
				return
			},
			EXIST: func(path string) (exists bool) {
				exists, _ = rscngepnt.fsexists(path)
				return
			},
			FINDROOT: func(path ...interface{}) (root string) {
				root, _ = rscngepnt.fsfindroot(path...)
				return
			},
			FINDROOTS: func(path ...interface{}) (roots []string) {
				roots, _ = rscngepnt.fsfindroots(path...)
				return
			},
			FIND: func(path ...interface{}) (finfos []fsutils.FileInfo) {
				finfos, _ = rscngepnt.fsfind(path...)
				return
			}, LS: func(path ...interface{}) (finfos []fsutils.FileInfo) {
				return
			}, MKDIR: func(path ...interface{}) bool {
				if len(path) == 1 {
					pth, _ := path[0].(string)
					return rscngepnt.fsmkdir(pth)
				}
				return false
			}, MKDIRALL: func(path ...interface{}) bool {
				if len(path) == 1 {
					pth, _ := path[0].(string)
					return rscngepnt.fsmkdirall(pth)
				}
				return false
			}, RM: func(path string) bool {
				return rscngepnt.fsrm(path)
			}, MV: func(path string, destpath string) bool {
				return rscngepnt.fsmv(path, destpath)
			}, TOUCH: func(path string) bool {
				return rscngepnt.fstouch(path)
			}, CAT: func(path string, a ...interface{}) io.Reader {
				return rscngepnt.fscat(path)
			}, CATS: func(path string, a ...interface{}) string {
				return rscngepnt.fscats(path, a...)
			}, SET: func(path string, a ...interface{}) bool {
				return rscngepnt.fsset(path, a...)
			}, APPEND: func(path string, a ...interface{}) bool {
				return rscngepnt.fsappend(path, a...)
			}, MULTICAT: func(path ...string) (r io.Reader) {
				return rscngepnt.multicat(path...)
			}, MULTICATS: func(path ...string) (s string) {
				return rscngepnt.multicats(path...)
			},
		}
	}
	return rscngepnt.fsutils
}

func isValidLocalPath(path string) bool {
	if fi, fierr := os.Stat(path); fi != nil && fierr == nil {
		return fi.IsDir()
	}
	return false
}

func (rscngepnt *ResourcingEndpoint) multicat(path ...string) (r io.Reader) {
	var rdrs []io.Reader = nil
	if pthl := len(path); pthl > 0 {
		rdrs = []io.Reader{}
		for _, pth := range path {
			if nxtr := rscngepnt.fscat(pth); nxtr != nil {
				rdrs = append(rdrs, nxtr)
			}
		}
	}
	r = iorw.NewMultiEOFCloseSeekReader(rdrs...)
	return
}

func (rscngepnt *ResourcingEndpoint) multicats(path ...string) (cntnt string) {
	if pthl := len(path); pthl > 0 {
		for _, pth := range path {
			if rs, _, _ := rscngepnt.findRS(pth); rs != nil {
				func() {
					defer rs.Close()
					if s, _ := iorw.ReaderToString(rs); s != "" {
						cntnt += s
					}
				}()
			}
		}
	}
	return
}

func (rscngepnt *ResourcingEndpoint) fsappend(path string, a ...interface{}) bool {
	if path = strings.Replace(strings.TrimSpace(path), "\\", "/", -1); path != "" && strings.LastIndex(path, ".") > 0 && (strings.LastIndex(path, "/") == -1 || strings.LastIndex(path, ".") > strings.LastIndex(path, "/")) {
		if rscngepnt.isLocal {
			if isValidLocalPath(rscngepnt.path) {
				if err := fsutils.APPEND(rscngepnt.path+path, a...); err == nil {
					return true
				} else {
					fmt.Println(err.Error())
				}
			}
		}
		if embdrs, emdrsok := rscngepnt.embeddedResources[path]; emdrsok {
			embdrs.Print(a...)
			return true
		} else {
			embdrs := NewEmbeddedResource(rscngepnt)
			embdrs.Print(a...)
			rscngepnt.embeddedResources[path] = embdrs
			return true
		}
	}
	return false
}

func (rscngepnt *ResourcingEndpoint) fsset(path string, a ...interface{}) bool {
	if path = strings.Replace(strings.TrimSpace(path), "\\", "/", -1); path != "" && strings.LastIndex(path, ".") > 0 && (strings.LastIndex(path, "/") == -1 || strings.LastIndex(path, ".") > strings.LastIndex(path, "/")) {
		if rscngepnt.isLocal {
			if isValidLocalPath(rscngepnt.path) {
				if err := fsutils.SET(rscngepnt.path+path, a...); err == nil {
					return true
				}
			}
		}
		if embdrs, emdrsok := rscngepnt.embeddedResources[path]; emdrsok {
			embdrs.Clear()
			embdrs.Print(a...)
			embdrs.path = path
			return true
		} else {
			embdrs := NewEmbeddedResource(rscngepnt)
			embdrs.Print(a...)
			embdrs.path = path
			rscngepnt.embeddedResources[path] = embdrs
			return true
		}
	}
	return false
}

func (rscngepnt *ResourcingEndpoint) fscat(path string, a ...interface{}) (r io.Reader) {
	if path = strings.Replace(strings.TrimSpace(path), "\\", "/", -1); path != "" && strings.LastIndex(path, ".") > 0 && (strings.LastIndex(path, "/") == -1 || strings.LastIndex(path, ".") > strings.LastIndex(path, "/")) {
		if rs, mdify, _ := rscngepnt.findRS(path); rs != nil {
			//if _, eofrsok := rs.(*iorw.EOFCloseSeekReader); eofrsok {
			//	r = rs
			//} else {
			//	r = iorw.NewEOFCloseSeekReader(rs)
			//}
			r = rs
			if len(a) > 0 {
				for _, d := range a {
					if d != nil {
						if fnrawOrActive, _ := d.(func(raw bool, active bool)); fnrawOrActive != nil {
							fnrawOrActive(rscngepnt.isRaw, rscngepnt.isActive)
						} else if fnmodified, _ := d.(func(modified time.Time)); fnmodified != nil {
							fnmodified(mdify)
						}
					}
				}
			}
		}
	}
	return r
}

func (rscngepnt *ResourcingEndpoint) fscats(path string, a ...interface{}) (s string) {
	if r := rscngepnt.fscat(path, a...); r != nil {
		s, _ = iorw.ReaderToString(r)
	}
	return s
}

func (rscngepnt *ResourcingEndpoint) fspipe(path string, a ...interface{}) (r io.Reader) {
	if path = strings.Replace(strings.TrimSpace(path), "\\", "/", -1); path != "" && strings.LastIndex(path, ".") > 0 && (strings.LastIndex(path, "/") == -1 || strings.LastIndex(path, ".") > strings.LastIndex(path, "/")) {
		if rs, mdify, _ := rscngepnt.findRS(path); rs != nil {
			r = iorw.NewEOFCloseSeekReader(rs, false)
			if len(a) > 0 {
				for _, d := range a {
					if d != nil {
						if fnrawOrActive, _ := d.(func(raw bool, active bool)); fnrawOrActive != nil {
							fnrawOrActive(rscngepnt.isRaw, rscngepnt.isActive)
							break
						} else if fnmodified, _ := d.(func(modified time.Time)); fnmodified != nil {
							fnmodified(mdify)
						}
					}
				}
			}
		}
	}
	return r
}

func (rscngepnt *ResourcingEndpoint) fspipes(path string, a ...interface{}) (s string) {
	if r := rscngepnt.fspipe(path, a...); r != nil {
		s, _ = iorw.ReaderToString(r)
	}
	return s
}

func (rscngepnt *ResourcingEndpoint) fstouch(path string) bool {
	if rscngepnt.isLocal {
		if path = strings.Replace(strings.TrimSpace(path), "\\", "/", -1); path != "" && strings.LastIndex(path, ".") > 0 && (strings.LastIndex(path, "/") == -1 || strings.LastIndex(path, ".") > strings.LastIndex(path, "/")) {
			if err := fsutils.TOUCH(rscngepnt.path + path); err != nil {
				return false
			}
		}
		return true
	}
	return false
}

func (rscngepnt *ResourcingEndpoint) fsmv(path string, destpath string) bool {
	if rscngepnt.isLocal {
		if path = strings.Replace(strings.TrimSpace(path), "\\", "/", -1); path != "" {
			if destpath = strings.Replace(strings.TrimSpace(destpath), "\\", "/", -1); destpath != "" {
				if err := fsutils.MV(rscngepnt.path+path, rscngepnt.path+destpath); err != nil {
					return false
				}
			}
		}
		return true
	}
	return false
}

func (rscngepnt *ResourcingEndpoint) fsrm(path string) (rmvd bool) {
	if rscngepnt.isLocal {
		if path = strings.Replace(strings.TrimSpace(path), "\\", "/", -1); path != "" {
			if err := fsutils.RM(rscngepnt.path + path); err == nil {
				rmvd = true
			}
		}
	}
	if !rmvd {
		if len(rscngepnt.embeddedResources) > 0 {
			for embdpth := range rscngepnt.embeddedResources {
				if strings.HasPrefix(embdpth, path) {
					rscngepnt.embeddedResources[embdpth].Close()
					rscngepnt.embeddedResources[embdpth] = nil
					delete(rscngepnt.embeddedResources, embdpth)
				}
			}
		}
	}
	return rmvd
}

func (rscngepnt *ResourcingEndpoint) fsmkdirall(path string) bool {
	if rscngepnt.isLocal {
		if path = strings.Replace(strings.TrimSpace(path), "\\", "/", -1); path != "" && (strings.LastIndex(path, ".") == -1 || strings.LastIndex(path, ".") < strings.LastIndex(path, "/")) {
			lklpath := rscngepnt.path + strings.TrimSpace(strings.Replace(path, "\\", "/", -1))
			if strings.LastIndex(lklpath, "/") > 0 && strings.HasSuffix(lklpath, "/") {
				lklpath = lklpath[:len(lklpath)-1]
			}
			if err := fsutils.MKDIRALL(lklpath); err != nil {
				return false
			}
		}
		return true
	}
	return false
}

func (rscngepnt *ResourcingEndpoint) fsmkdir(path string) bool {
	if rscngepnt.isLocal {
		if path = strings.Replace(strings.TrimSpace(path), "\\", "/", -1); path != "" && (strings.LastIndex(path, ".") == -1 || strings.LastIndex(path, ".") < strings.LastIndex(path, "/")) {
			lklpath := rscngepnt.path + strings.TrimSpace(strings.Replace(path, "\\", "/", -1))
			if strings.LastIndex(lklpath, "/") > 0 && strings.HasSuffix(lklpath, "/") {
				lklpath = lklpath[:len(lklpath)-1]
			}
			if err := fsutils.MKDIR(lklpath); err != nil {
				return false
			}
		}
		return true
	}
	return false
}

func (rscngepnt *ResourcingEndpoint) fsabs(path ...string) (abspath string, err error) {
	rsroot := rscngepnt.rsngmngr.rsngendpaths[rscngepnt]
	if rscngepnt.isLocal {
		lklpath := rscngepnt.path + strings.TrimSpace(strings.Replace(path[0], "\\", "/", -1))
		if strings.LastIndex(lklpath, "/") > 0 && strings.HasSuffix(lklpath, "/") {
			lklpath = lklpath[:len(lklpath)-1]
		}
		if len(path) == 1 {
			if path[0] != "" {
				path[0] = strings.Replace(path[0], "\\", "/", -1)
			}
			abspath, err = fsutils.ABS(lklpath + strings.TrimSpace(strings.Replace(path[0], "\\", "/", -1)))
			return
		} else if len(path) == 2 {
			if path[1] != "" {
				path[1] = strings.Replace(path[1], "\\", "/", -1)
			}
			abspath, err = fsutils.ABS(lklpath + strings.TrimSpace(strings.Replace(path[1], "\\", "/", -1)))
			return
		}
	}
	if rscngepnt.embeddedResources != nil {
		if pthl := len(path); pthl > 0 {
			for embdrspth, emdbrs := range rscngepnt.embeddedResources {
				if strings.HasPrefix(embdrspth, path[0]) && (embdrspth == path[0] || path[0] == "" && strings.LastIndex(embdrspth, "/") == -1 && strings.LastIndex(embdrspth, "/") < strings.LastIndex(embdrspth, ".")) {
					lkppath := embdrspth
					if pthl == 1 {
						if finfo := fsutils.NewFSUtils().DUMMYFINFO(emdbrs.Name(), lkppath, lkppath, rsroot, emdbrs.Size(), 0, emdbrs.modified, rscngepnt.isActive, rscngepnt.isRaw, emdbrs.fsopener); finfo != nil {
							abspath = finfo.AbsolutePath()
							finfo = nil
							break
						}
					} else if pthl == 2 {
						if path[1] != "" {
							path[1] = strings.Replace(path[0], "\\", "/", -1)
						}
						if path[0] == "" {
							lkppath = path[1] + "/" + lkppath
						} else {
							lkppath = path[1][:len(path[1])-len(embdrspth)] + embdrspth
						}
						if finfo := fsutils.NewFSUtils().DUMMYFINFO(emdbrs.Name(), lkppath, lkppath, rsroot, emdbrs.Size(), 0, emdbrs.modified, rscngepnt.isActive, rscngepnt.isRaw, emdbrs.fsopener); finfo != nil {
							abspath = finfo.AbsolutePath()
							finfo = nil
							break
						}
					}
				}
			}
		}
	}
	return
}

func checkPathMask(path string, mask string) (vld bool) {
	vld, _ = filepath.Match(mask, path)
	return
}

var lkzpextsdefualtfunc = func(pzext, path, curntpzpath, rmndrpath string, fnrd ...func(fi fs.FileInfo, f io.ReadCloser, ferr error)) (lsroot fs.FileInfo, lclfsinfo []fs.FileInfo) {
	if r, rerr := OpenReader(curntpzpath); rerr == nil {
		chpthmsk := false
		if path != "" && strings.ContainsAny(path, "*.?") {
			chpthmsk = strings.ContainsAny(path, "*?")
		}
		var tarr *tar.Reader = nil
		var gzr *gzip.Reader
		var gzrerr error = nil
		defer func() {
			if len(fnrd) == 0 || fnrd[0] == nil {
				if tarr != nil {
					tarr = nil
				}
				if gzr != nil {
					gzr.Close()
				}
				if r != nil {
					r.Close()
				}
			}
		}()
		if pzext == ".tgz" || pzext == ".gz" {
			gzr, gzrerr = gzip.NewReader(r)
			if gzrerr != nil {
				return
			}
			if gzr != nil {
				tarr = tar.NewReader(iorw.NewEOFCloseSeekReader(gzr, true))
			}
		} else if pzext == ".tar" {
			tarr = tar.NewReader(r)
		}
		if tarr != nil {
			rmndrpthl := len(rmndrpath)
			for {
				trhead, trerr := tarr.Next()
				if trerr == io.EOF {
					break
				} else if trerr != nil {
					break
				}
				if trhead != nil {
					switch trhead.Typeflag {
					case tar.TypeReg:
						fpath, fpathl := trhead.Name, len(trhead.Name)
						if fpathl >= rmndrpthl && fpath[:rmndrpthl] == rmndrpath {
							finfo := trhead.FileInfo()
							if path != "" {
								if chpthmsk {
									if checkPathMask(fpath[rmndrpthl:], path) {
										lclfsinfo = append(lclfsinfo, finfo)
										continue
									}
									continue
								}
								if path == fpath[rmndrpthl:] {
									if len(fnrd) == 1 && fnrd[0] != nil {
										rdr := iorw.ReadFunc(tarr.Read)
										f := iorw.NewEOFCloseSeekReader(rdr, true)
										fnrd[0](finfo, f, nil)
										fnrd[0] = nil
										if tarr != nil {
											tarr = nil
										}
										if gzr != nil {
											gzr = nil
										}
										return
									}
									lclfsinfo = append(lclfsinfo, finfo)
									return
								}
								continue
							}
							if finfo.IsDir() {
								lsroot = finfo
								return
							}
						}
					}
				} else {
					continue
				}
			}
		}
	}
	return
}
var lklzpextsfunc = map[string]func(path, curntpzpath, rmndrpath string, fnrd ...func(fi fs.FileInfo, f io.ReadCloser, ferr error)) (lsroot fs.FileInfo, lclfsinfo []fs.FileInfo){".zip": func(path, curntpzpath, rmndrpath string, fnrd ...func(fi fs.FileInfo, f io.ReadCloser, ferr error)) (lsroot fs.FileInfo, lclfsinfo []fs.FileInfo) {
	rmndrpthl := len(rmndrpath)
	if rmndrpthl > 0 && rmndrpath[rmndrpthl-1] != '/' {
		rmndrpath += "/"
		rmndrpthl++
	}
	zpr, zprerr := zip.OpenReader(curntpzpath)
	if func() bool {
		defer func() {
			if len(fnrd) == 0 || fnrd[0] == nil {
				zpr.Close()
			}
		}()

		if zprerr != nil {
			return true
		}
		zpfls := zpr.File
		var iterzp = func(yield func(*zip.File) bool) {
			lzp := len(zpfls)
			chpthmsk := false
			if path != "" && strings.ContainsAny(path, "*.?") {
				chpthmsk = strings.ContainsAny(path, "*?")
			}
			for lzp > 0 {
				zf := zpfls[0]

				zpfls = zpfls[1:]
				lzp--
				zfnml, zfnm := len(zf.Name), zf.Name
				if zfnml >= rmndrpthl && zfnm[:rmndrpthl] == rmndrpath {
					tstnme := zf.Name[rmndrpthl:]
					zpthi := strings.Index(tstnme, "/")
					if (zpthi > -1 && tstnme == tstnme[:zpthi+1]) || zpthi == -1 {
						if chpthmsk {
							if checkPathMask(tstnme, path) {
								if !yield(zf) {
									return
								}
								continue
							}
							continue
						}
						if path == "" {
							if tstnme == "" {
								if !yield(zf) {
									return
								}
								return
							}
							continue
						}
						if tstnme != "" && tstnme == path {
							if !yield(zf) {
								return
							}
							return
						}
					}
				}
			}
		}
		for zf := range iterzp {
			zfi := zf.FileHeader.FileInfo()
			if zfi != nil {
				if zfi.IsDir() {
					if path == "" {
						lsroot = zfi
						return true
					}
				}
				if len(fnrd) == 1 && fnrd[0] != nil {
					f, ferr := zf.Open()
					fnrd[0](zfi, f, ferr)
					if ferr == nil {
						return true
					}
					fnrd[0] = nil
					return true
				}
				lclfsinfo = append(lclfsinfo, zfi)
				continue
			}
		}
		return false
	}() {
		return
	}
	return
}}

func localCompressLs(pathroot, pzpext, path string, fnrd ...func(fi fs.FileInfo, f io.ReadCloser, err error)) (lsroot fs.FileInfo, lclfsinfo []fs.FileInfo) {
	lstpzi := 0

	if nxtpzi := strings.Index(pathroot[lstpzi:], pzpext); nxtpzi > 0 {
		if curntpzpath := pathroot[lstpzi : nxtpzi+len(pzpext)]; curntpzpath != "" {
			rmndrpath := pathroot[nxtpzi+len(pzpext)+1:]
			if nxtfsfnk := lklzpextsfunc[pzpext]; nxtfsfnk != nil {
				lsroot, lclfsinfo = nxtfsfnk(path, curntpzpath, rmndrpath, fnrd...)
				return
			}
			lsroot, lclfsinfo = lkzpextsdefualtfunc(pzpext, path, curntpzpath, rmndrpath, fnrd...)
			/*if pzpext == ".zip" {
				rmndrpthl := len(rmndrpath)
				zpr, zprerr := zip.OpenReader(curntpzpath)
				if func() bool {
					defer func() {
						if len(fnrd) == 0 || fnrd[0] == nil {
							zpr.Close()
						}
					}()

					if zprerr != nil {
						return true
					}
					zpfls := zpr.File
					var iterzp = func(yield func(*zip.File) bool) {
						lzp := len(zpfls)
						chpthmsk := false
						if path != "" && strings.ContainsAny(path, "*.?") {
							chpthmsk = strings.ContainsAny(path, "*?")
						}
						for lzp > 0 {
							zf := zpfls[0]

							zpfls = zpfls[1:]
							lzp--
							zfnml, zfnm := len(zf.Name), zf.Name
							if zfnml >= rmndrpthl && zfnm[:rmndrpthl] == rmndrpath {
								tstnme := zf.Name[rmndrpthl:]
								zpthi := strings.Index(tstnme, "/")
								if (zpthi > -1 && tstnme == tstnme[:zpthi+1]) || zpthi == -1 {
									if chpthmsk {
										if checkPathMask(tstnme, path) {
											if !yield(zf) {
												return
											}
											continue
										}
										continue
									}
									if path == "" {
										if tstnme == "" {
											if !yield(zf) {
												return
											}
											return
										}
										continue
									}
									if tstnme != "" && tstnme == path {
										if !yield(zf) {
											return
										}
										return
									}
								}
							}
						}
					}
					for zf := range iterzp {
						zfi := zf.FileHeader.FileInfo()
						if zfi != nil {
							if zfi.IsDir() {
								if path == "" {
									lsroot = zfi
									return true
								}
							}
							if len(fnrd) == 1 && fnrd[0] != nil {
								f, ferr := zf.Open()
								fnrd[0](zfi, f, ferr)
								if ferr == nil {
									return true
								}
								fnrd[0] = nil
								return true
							}
							lclfsinfo = append(lclfsinfo, zfi)
							continue
						}
					}
					return false
				}() {
					return
				}
			}*/
			lstpzi = nxtpzi + len(pzpext)
		}

	}
	return

}

func localLs(pathroot, path string, fnrd ...func(fi fs.FileInfo, f io.ReadCloser, err error)) (lsroot fs.FileInfo, lclfsinfo []fs.FileInfo) {
	if lkpzpi, lkpzpext := lkpzpextindex(pathroot); lkpzpi > 0 {
		return localCompressLs(pathroot, lkpzpext, path, fnrd...)
	}

	if path != "" {
		chpthmsk := false
		if path != "" && strings.ContainsAny(path, "*.?") {
			chpthmsk = strings.ContainsAny(path, "*?")
		}

		if path != "" && path[0] == '/' {
			path = path[1:]
			if pathroot != "" && pathroot[len(pathroot)-1] != '/' {
				pathroot += "/"
			}
		}
		if f, ferr := os.Open(pathroot); f != nil && ferr == nil {
			defer func() {
				if len(fnrd) > 0 && fnrd[0] != nil {
					fnrd[0] = nil
				}
				f.Close()
			}()
			if flclfinfos, _ := f.Readdir(0); len(flclfinfos) > 0 {
				for _, flclfi := range flclfinfos {
					if chpthmsk {
						if checkPathMask(flclfi.Name(), path) {
							lclfsinfo = append(lclfsinfo, flclfi)
						}
						continue
					}
					if path != "" {
						if path == flclfi.Name() {
							if len(fnrd) > 0 && fnrd[0] != nil {
								f, ferr := os.Open(pathroot + path)
								if ferr == nil {
									fnrd[0](flclfi, f, ferr)
									return
								}
								return
							}
							lclfsinfo = append(lclfsinfo, flclfi)
							return
						}
						continue
					}
				}
			}
			return
		}
		return
	}
	if fi, _ := os.Stat(pathroot + path); fi != nil {
		if fi.IsDir() {
			if path == "" {
				lsroot = fi
				return
			}
		}
		lclfsinfo = append(lclfsinfo, fi)
	}
	return
}

func (rscngepnt *ResourcingEndpoint) fsls(paths ...interface{}) (finfos []fsutils.FileInfo) {
	rsroot := rscngepnt.rsngmngr.rsngendpaths[rscngepnt]
	path := []string{}
	a := []interface{}{}
	for _, d := range paths {
		if ds, dsk := d.(string); dsk {
			path = append(path, ds)
		} else {
			a = append(a, d)
		}
	}
	var addpth = rscngepnt.rsngmngr.rsngendpaths[rscngepnt]
	if addpth != "" {
		addpth = addpth + ""
		if !strings.HasSuffix(addpth, "/") {
			addpth += "/"
		}
	}
	if rscngepnt.isLocal {
		subpath := strings.TrimFunc(strings.Replace(path[0], "\\", "/", -1), iorw.IsSpace)
		subroot := ""
		if subi := strings.LastIndex(subpath, "/"); subi > -1 {
			subroot = subpath[:subi+1]
			subpath = subpath[subi+1:]
		}
		subdone := map[string]bool{}
		for _, subpth := range strings.Split(subpath, ",") {
			subpth := strings.TrimFunc(subpth, iorw.IsSpace)
			if subdone[subpth] {
				continue
			}
			subdone[subpth] = true

			lclroot, lclfsinfos := localLs(func() string {
				if subroot != "" && rscngepnt.path[len(rscngepnt.path)-1] != '/' && subroot[0] != '/' {
					return rscngepnt.path + "/" + subroot
				}
				return rscngepnt.path + subroot
			}(), subpth)
			if lclroot != nil {
				if subroot == "" {
					if rsroot != "" && rsroot[len(rsroot)-1] != '/' {
						subroot = "/"
					}
				} else if rsroot != "" && rsroot[len(rsroot)-1] != '/' && subroot[0] != '/' {
					subroot = "/" + subroot
				}

				finfos = append(finfos, fsutils.DUMMYFINFO("", rsroot+subroot, rsroot+subroot, rsroot, lclroot.Size(), lclroot.Mode(), lclroot.ModTime(), rscngepnt.isActive, rscngepnt.isRaw, rscngepnt.fsopener))
			}
			for _, lclfin := range lclfsinfos {
				if lclfin != nil {
					if lclfin.IsDir() {
						if subroot == "" {
							if rsroot != "" && rsroot[len(rsroot)-1] != '/' {
								subroot = "/"
							}
						} else if rsroot != "" && rsroot[len(rsroot)-1] != '/' && subroot[0] != '/' {
							subroot = "/" + subroot
						}
						finfos = append(finfos, fsutils.DUMMYFINFO(lclfin.Name(), rsroot+subroot+lclfin.Name()+"/", rsroot+subroot+lclfin.Name()+"/", rsroot, lclfin.Size(), lclfin.Mode(), lclfin.ModTime(), rscngepnt.isActive, rscngepnt.isRaw, rscngepnt.fsopener))
						continue
					}
					if subroot == "" {
						if rsroot != "" && rsroot[len(rsroot)-1] != '/' {
							subroot = "/"
						}
					} else if rsroot != "" && rsroot[len(rsroot)-1] != '/' && subroot[0] != '/' {
						subroot = "/" + subroot
					}
					finfos = append(finfos, fsutils.DUMMYFINFO(lclfin.Name(), rsroot+subroot+lclfin.Name(), rsroot+subroot+lclfin.Name(), rsroot, lclfin.Size(), lclfin.Mode(), lclfin.ModTime(), rscngepnt.isActive, rscngepnt.isRaw, rscngepnt.fsopener))
				}
			}
		}
	}
	if rscngepnt.embeddedResources != nil {
		if pthl := len(path); pthl > 0 {
			if pthl == 1 && path[0] == "" && len(rscngepnt.embeddedResources) > 0 {
				if len(finfos) == 0 {
					finfos = append(finfos, fsutils.DUMMYFINFO("", rsroot+"/", rsroot+"/", rsroot+"/", 0, os.ModeDir, time.Now(), rscngepnt.isActive, rscngepnt.isRaw, rscngepnt.fsopener))
				}
			} else {
				for embdrspth, emdbrs := range rscngepnt.embeddedResources {
					if strings.HasPrefix(embdrspth, path[0]) && (embdrspth == path[0] || path[0] == "" && strings.LastIndex(embdrspth, "/") == -1 && strings.LastIndex(embdrspth, "/") < strings.LastIndex(embdrspth, ".")) {
						lkppath := embdrspth
						if pthl == 1 {
							finfos = append(finfos, fsutils.NewFSUtils().DUMMYFINFO(emdbrs.Name(), addpth+lkppath, addpth+lkppath, rsroot, emdbrs.Size(), 0, emdbrs.modified, rscngepnt.isActive, rscngepnt.isRaw, emdbrs.fsopener))
						} else if pthl == 2 {
							if path[0] == "" {
								if strings.HasSuffix(path[1], "/") {
									lkppath = path[1][:len(path[1])-1] + "/" + lkppath
								} else {
									lkppath = path[1] + "/" + lkppath
								}
							} else {
								lkppath = path[1][:len(path[1])-len(embdrspth)] + embdrspth
							}
							finfos = append(finfos, fsutils.DUMMYFINFO(emdbrs.Name(), addpth+lkppath, addpth+lkppath, rsroot, emdbrs.Size(), 0, emdbrs.modified, rscngepnt.isActive, rscngepnt.isRaw, emdbrs.fsopener))
						}
					}
				}
			}
		}
	}
	if len(a) > 0 {
		/*for _, d := range a {
			if d != nil {
				if fnrawOrActive, _ := d.(func(raw bool, active bool)); fnrawOrActive != nil {
					fnrawOrActive(rscngepnt.isRaw, rscngepnt.isActive)
				} else if fnmodified, _ := d.(func(modified time.Time)); fnmodified != nil {
					if len(finfos) == 1 {
						fnmodified(finfos[0].ModTime())
					}
				}
			}
		}*/
	}
	return
}

func (rscngepnt *ResourcingEndpoint) fsexists(path string) (pathexists bool, err error) {
	if lsinfo := rscngepnt.fsls(path); len(lsinfo) == 1 {
		pathexists = true
	}
	return
}

func (rscngepnt *ResourcingEndpoint) fsfindroot(path ...interface{}) (root string, err error) {
	var roots []string = nil
	if roots, err = rscngepnt.fsfindroots(path...); err == nil && len(roots) > 0 {
		root = roots[0]
	}
	roots = nil
	return
}

func (rscngepnt *ResourcingEndpoint) fsfindroots(paths ...interface{}) (roots []string, err error) {
	path := []string{}
	a := []interface{}{}

	for _, d := range paths {
		if ds, dsk := d.(string); dsk {
			path = append(path, ds)
			a = append(a, ds)
		} else {
			a = append(a, d)
		}
	}
	if fios, fioserr := rscngepnt.fsfind(a...); fioserr == nil && len(fios) > 0 {
		pathsfound := []string{}
		maxlen := 0
		for _, fio := range fios {
			if fio.IsDir() {
				if fiopath := fio.Path(); strings.HasPrefix(fiopath, path[0]) {
					if len(fiopath) > maxlen {
						pathsfound = append(pathsfound, fiopath)
						maxlen = len(fiopath)
					}
				}
			}
		}
		for _, pthsfnd := range pathsfound {
			if len(pthsfnd) == maxlen {
				if len(path) > 1 {
					roots = append(roots, path[1]+pthsfnd[len(path[0]):])
				} else {
					roots = append(roots, pthsfnd)
				}
			}
		}
	} else {
		err = fioserr
	}
	return
}

func (rscngepnt *ResourcingEndpoint) fsopener(path string, a ...interface{}) (r io.ReadCloser, err error) {
	if rscngepnt != nil && path != "" {
		var rmod time.Time
		if r, rmod, err = rscngepnt.findRS(path); r != nil && err == nil {
			if len(a) > 0 {
				for _, d := range a {
					if d != nil {
						if fnrawOrActive, _ := d.(func(raw bool, active bool)); fnrawOrActive != nil {
							fnrawOrActive(rscngepnt.isRaw, rscngepnt.isActive)
						} else if fnmodified, _ := d.(func(modified time.Time)); fnmodified != nil {
							fnmodified(rmod)
						}
					}
				}
			}
		}
	}
	return
}

func (rscngepnt *ResourcingEndpoint) fsfind(paths ...interface{}) (finfos []fsutils.FileInfo, err error) {
	rsroot := rscngepnt.rsngmngr.rsngendpaths[rscngepnt]
	path := []string{}
	a := []interface{}{}
	for _, d := range paths {
		if ds, dsk := d.(string); dsk {
			path = append(path, ds)
		} else {
			a = append(a, d)
		}
	}
	var addpth = rscngepnt.rsngmngr.rsngendpaths[rscngepnt]
	if addpth != "" {
		addpth = addpth + ""
		if !strings.HasSuffix(addpth, "/") {
			addpth += "/"
		}
	}
	lklpath := rscngepnt.path + strings.TrimSpace(strings.Replace(path[0], "\\", "/", -1))
	if strings.LastIndex(lklpath, "/") > 0 && strings.HasSuffix(lklpath, "/") {
		lklpath = lklpath[:len(lklpath)-1]
	}
	if rscngepnt.isLocal {
		if len(path) == 1 {
			finfos, _ = fsutils.FIND(lklpath, append([]interface{}{strings.TrimSpace(strings.Replace(addpth+path[0], "\\", "/", -1)), rscngepnt.fsopener}, a...)...)
		} else if len(path) == 2 {
			if strings.HasPrefix(path[1], addpth) {
				path[1] = path[1][len(addpth):]
			}
			finfos, _ = fsutils.FIND(lklpath, append([]interface{}{strings.TrimSpace(strings.Replace(addpth+path[1], "\\", "/", -1)), rscngepnt.fsopener}, a...)...)
		}
	}
	if rscngepnt.embeddedResources != nil {
		if pthl := len(path); pthl > 0 {
			for embdrspth, emdbrs := range rscngepnt.embeddedResources {
				if finfos == nil {
					finfos = []fsutils.FileInfo{}
				}
				if strings.HasPrefix(embdrspth, path[0]) && (embdrspth == path[0] || path[0] == "" && strings.LastIndex(embdrspth, "/") == -1 && strings.LastIndex(embdrspth, "/") < strings.LastIndex(embdrspth, ".")) {
					lkppath := embdrspth
					if pthl == 1 {
						finfos = append(finfos, fsutils.NewFSUtils().DUMMYFINFO(emdbrs.Name(), addpth+lkppath, addpth+lkppath, rsroot, emdbrs.Size(), 0, emdbrs.modified, rscngepnt.isActive, rscngepnt.isRaw, emdbrs.fsopener))
					} else if pthl == 2 {
						if path[0] == "" {
							lkppath = path[1] + "/" + lkppath
						} else {
							lkppath = path[1][:len(path[1])-len(embdrspth)] + embdrspth
						}
						if strings.HasPrefix(lkppath, addpth) {
							lkppath = lkppath[len(addpth):]
						}
						finfos = append(finfos, fsutils.NewFSUtils().DUMMYFINFO(emdbrs.Name(), addpth+lkppath, addpth+lkppath, rsroot, emdbrs.Size(), 0, emdbrs.modified, rscngepnt.isActive, rscngepnt.isRaw, emdbrs.fsopener))
					}
				}
			}
		}
	}
	if len(a) > 0 {
		/*for _, d := range a {
			if d != nil {
				if fnrawOrActive, _ := d.(func(raw bool, active bool)); fnrawOrActive != nil {
					fnrawOrActive(rscngepnt.isRaw, rscngepnt.isActive)
				} else if fnmodified, _ := d.(func(modified time.Time)); fnmodified != nil {
					if len(finfos) == 1 {
						fnmodified(finfos[0].ModTime())
					}
				}
			}
		}*/
	}
	return
}

func (rscngepnt *ResourcingEndpoint) dispose() {
	if rscngepnt != nil {
		if rsngmngr := rscngepnt.rsngmngr; rsngmngr != nil {
			rscngepnt.rsngmngr = nil
			/*rsendpath := rscngepnt.path
			delete(rscngepnt.rsngmngr.rsngrootpaths, rsendpath)
			for rspth, rsndpth := range rscngepnt.rsngmngr.rsngpaths {
				if rsndpth == rsendpath {
					delete(rscngepnt.rsngmngr.rsngpaths, rspth)
				}
			}*/
			if epntpth, epntpthok := rsngmngr.rsngendpaths[rscngepnt]; epntpthok {
				delete(rsngmngr.rsngendpaths, rscngepnt)
				delete(rsngmngr.rsngpaths, epntpth)
			}
			rscngepnt.rsngmngr = nil
		}
		if rscngepnt.embeddedResources != nil {
			for embk := range rscngepnt.embeddedResources {
				rscngepnt.RemoveResource(embk)
			}
			rscngepnt.embeddedResources = nil
		}
		if rscngepnt.fsutils != nil {
			rscngepnt.fsutils = nil
		}
		rscngepnt = nil
	}
}

var lklzpexts = map[string]bool{".zip": true, ".tgz": true, ".gz": true, ".tar": true, ".jar": true, ".war": true}

func lkpzpextindex(lkppath string) (lkpi int, lkpext string) {
	for lkpk := range lklzpexts {
		if lkpi = strings.Index(lkppath, lkpk); lkpi > -1 {
			lkpext = lkpk
			break
		}
	}
	return
}

func OpenReader(name string) (*iorw.EOFCloseSeekReader, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	} else if fi.Size() > 0 {
		return iorw.NewEOFCloseSeekReader(f, true), nil
	}
	return nil, nil
}

func getLocalResource(lklpath string, path string, cachableExtsBuffs *iocaching.BufferCache, fsNotifyEvent func(path string, modified time.Time)) (rs io.ReadCloser, modified time.Time, err error) {
	lkpzpi, lkpzpext := lkpzpextindex(lklpath)
	pthzpi, _ := lkpzpextindex(path)

	if lkpzpi > -1 || pthzpi > -1 {
		orgpath := path
		if pthspi := strings.LastIndex(path, "/"); pthspi > -1 {
			if lklpath != "" {
				if path[0] == '/' {
					if lklpath[len(lklpath)-1] == '/' {
						lklpath += path[1 : pthspi+1]
					} else {
						lklpath += path[:pthspi+1]
					}
				} else {
					if lklpath[len(lklpath)-1] == '/' {
						lklpath += path[:pthspi+1]
					} else {
						lklpath += "/" + path[:pthspi+1]
					}
				}
				path = path[pthspi+1:]
			}
		}
		if path != "" {
			localCompressLs(lklpath, lkpzpext, path, func(zfi fs.FileInfo, zf io.ReadCloser, ferr error) {
				if ferr != nil {
					return
				}
				modified = zfi.ModTime()
				if cachableExtsBuffs != nil && cachableExts[filepath.Ext(orgpath)] {
					if bufr, bofmod := cachableExtsBuffs.Reader(orgpath); bufr == nil || (modified != bofmod) {
						func() {
							defer zf.Close()
							if prebf, prebferr := iorw.NewBufferError(zf); prebferr == nil {
								cachableExtsBuffs.Set(orgpath, modified, prebf.Reader())
								if fsNotifyEvent != nil {
									go fsNotifyEvent(orgpath, modified)
								}
							}
						}()
						rs, _ = cachableExtsBuffs.Reader(orgpath)
					} else {
						rs = bufr
					}
				} else {
					rs = zf
				}
			})
			path = orgpath
		}
		/*var tmppath = ""
		var zpext = ""
		var lklpth = ""
		var tmppaths = strings.Split(func() (pathtouse string) {
			if lkpzpi > -1 && pthzpi == -1 {
				zpext = lkpzpext
				if strings.HasSuffix(lklpath, "/") {
					pathtouse = lklpath + path
				} else {
					pathtouse = lklpath + "/" + path
				}
			} else if lkpzpi == -1 && pthzpi > -1 {
				zpext = pthzpext
				if strings.HasSuffix(lklpath, "/") {
					pathtouse = lklpath + path
				} else {
					pathtouse = lklpath + "/" + path
				}
			}
			if zpext != "" {
				if zpi := strings.Index(pathtouse, zpext); zpi > -1 {
					lklpth = pathtouse[:strings.LastIndex(pathtouse[:zpi], "/")+1]
					pathtouse = pathtouse[len(lklpth):]
					pathtouse = strings.Replace(pathtouse, zpext, "", 1)
				}
			}
			return
		}(), "/")
		for pn, ps := range tmppaths {
			if tmpl := len(tmppaths); pn < tmpl-1 {
				if strings.HasPrefix(tmppath, "/") && strings.HasSuffix(lklpth, "/") {
					tmppath = tmppath[1:]
				}
				if _, fierr := os.Stat(lklpth + tmppath + ps + zpext); fierr == nil {
					var testpath = strings.Join(tmppaths[pn+1:tmpl], "/")
					if testpath != "" {
						if zpext == ".gz" || zpext == ".tgz" || zpext == ".tar" {
							if r, rerr := OpenReader(lklpth + tmppath + ps + zpext); rerr == nil {
								var tarr *tar.Reader = nil
								if zpext == ".tgz" || zpext == ".gz" {
									if gzr, gzrerr := gzip.NewReader(r); gzrerr != nil {
										continue
									} else {
										tarr = tar.NewReader(iorw.NewEOFCloseSeekReader(gzr, true))
									}
								} else if zpext == ".tar" {
									tarr = tar.NewReader(r)
								}
								if tarr != nil {
									for {
										trhead, trerr := tarr.Next()
										if trerr == io.EOF {
											break
										} else if trerr != nil {
											err = trerr
											break
										}
										if trhead != nil {
											switch trhead.Typeflag {
											case tar.TypeReg:
												fpath := trhead.Name
												finfo := trhead.FileInfo()
												if !finfo.IsDir() && fpath == testpath {
													modified = trhead.ModTime
													if cachableExtsBuffs != nil && cachableExts[path] {
														if bufr, bofmod := cachableExtsBuffs.Reader(path); bufr == nil || (modified != bofmod) {
															cachableExtsBuffs.Set(path, modified, trerr)
															rs, _ = cachableExtsBuffs.Reader(path)
															if fsNotifyEvent != nil {
																go fsNotifyEvent(path, modified)
															}
														} else {
															rs = bufr
														}
													} else {
														rs = iorw.NewEOFCloseSeekReader(tarr, true)
													}
													return
												}
											}
										} else {
											continue
										}
									}
								}
							}
						} else {
							if r, rerr := zip.OpenReader(lklpth + tmppath + ps + zpext); rerr == nil {
								for _, f := range r.File {
									if f.Name == testpath {
										modified = f.Modified
										if cachableExtsBuffs != nil && cachableExts[path] {
											if bufr, bofmod := cachableExtsBuffs.Reader(path); bufr == nil || (modified != bofmod) {
												if rc, rcerr := f.Open(); rcerr == nil {
													func() {
														defer rc.Close()
														if prebf, prebferr := iorw.NewBufferError(rc); prebferr == nil {
															cachableExtsBuffs.Set(path, modified, prebf.Reader())
															if fsNotifyEvent != nil {
																go fsNotifyEvent(path, modified)
															}
														}
													}()
													rs, _ = cachableExtsBuffs.Reader(path)

												} else {
													err = rcerr
												}
											} else if bufr != nil {
												rs = bufr
											}
										} else if rc, rcerr := f.Open(); rcerr == nil {
											rs = rc
										} else {
											err = rcerr
										}
										return
									}
								}
							}
						}
					}
					break
				} else {
					tmppath = tmppath + ps + "/"
				}
			} else {
				break
			}
		}*/
	} else if lkpzpi == -1 {
		orgpath := path
		if pthspi := strings.LastIndex(path, "/"); pthspi > -1 {
			if lklpath != "" {
				if path[0] == '/' {
					if lklpath[len(lklpath)-1] == '/' {
						lklpath += path[1 : pthspi+1]
					} else {
						lklpath += path[:pthspi+1]
					}
				} else {
					if lklpath[len(lklpath)-1] == '/' {
						lklpath += path[:pthspi+1]
					} else {
						lklpath += "/" + path[:pthspi+1]
					}
				}
				path = path[pthspi+1:]
			}
		}
		localLs(lklpath, path, func(fi fs.FileInfo, f io.ReadCloser, ferr error) {
			if !fi.IsDir() {
				modified = fi.ModTime()
				if cachableExtsBuffs != nil && cachableExts[filepath.Ext(path)] {
					if bufr, bufmod := cachableExtsBuffs.Reader(path); bufr == nil || (bufmod != fi.ModTime()) {
						if f != nil {
							func() {
								defer f.Close()
								if prebf, prebferr := iorw.NewBufferError(f); prebferr == nil {
									cachableExtsBuffs.Set(path, fi.ModTime(), prebf.Reader())
									if fsNotifyEvent != nil {
										go fsNotifyEvent(path, modified)
									}
								}
							}()
							rs, _ = cachableExtsBuffs.Reader(path)
						}
					} else {
						rs = bufr
					}
				} else if ferr == nil {
					rs = f
				} else if ferr != nil {
					err = ferr
				}
			} else if f != nil {
				f.Close()
			}
		})
		path = orgpath
	}
	if rs == nil && cachableExtsBuffs != nil && path != "" {
		cachableExtsBuffs.Del(path)
	}
	return
}

func (rscngepnt *ResourcingEndpoint) findRS(path string) (rs io.ReadCloser, modified time.Time, err error) {
	if path != "" {
		func() {
			if path = strings.TrimSpace(strings.Replace(path, "\\", "/", -1)); path != "" {
				embedpath := path
				if embedpath != "" && embedpath[1] == '/' {
					embedpath = embedpath[1:]
				}
				if rscngepnt.fs != nil {
					fspath := rscngepnt.path
					fspath = strings.TrimPrefix(fspath, "/")
					if strings.Contains(fspath, "/") {
						fspath = fspath[strings.Index(fspath, "/")+1:]
					}
					if fspath != "" && !strings.HasSuffix(fspath, "/") {
						fspath += "/"
					}

					if fsrs, _ := rscngepnt.fs.Open(fspath + path); fsrs != nil {
						if rscngepnt.isLocal && rscngepnt.cachableExtsBuffs != nil && cachableExts[filepath.Ext(path)] {
							if fis, fiserr := os.Stat(path); fiserr == nil {
								rs = rscngepnt.cachableExtsBuffs.ReaderReplace(path, fis.ModTime(), fsrs)
							} else {
								if rc, _ := fsrs.(io.ReadCloser); rc != nil {
									rs = rc
								} else {
									rs = iorw.NewEOFCloseSeekReader(fsrs)
								}
							}
						} else {
							if rc, _ := fsrs.(io.ReadCloser); rc != nil {
								rs = rc
							} else {
								rs = iorw.NewEOFCloseSeekReader(fsrs)
							}
						}
						return
					}
				}
				if embdrs, embdrsok := rscngepnt.embeddedResources[embedpath]; embdrsok {
					if embdrs != nil {
						modified = embdrs.modified
						rs = embdrs.Reader()
					}
				} else if rscngepnt.isLocal {
					if apppath := rscngepnt.rsngmngr.rsngendpaths[rscngepnt]; apppath != "" && strings.HasPrefix(path, apppath) {
						path = path[len(apppath):]
					}
					rs, modified, err = getLocalResource(rscngepnt.path, path, rscngepnt.cachableExtsBuffs, rscngepnt.rsngmngr.fsNotifyEvent)
				} else if rscngepnt.isRemote {
					prms := map[string]interface{}{}
					if rscngepnt.querystring != "" {
						if strings.LastIndex(path, "?") > 0 && (strings.LastIndex(path, "/") == -1 || strings.LastIndex(path, "?") > strings.LastIndex(path, "/")) {
							path += "&" + rscngepnt.querystring
						} else {
							path += "?" + rscngepnt.querystring
						}
					}
					remoteHeaders := map[string]string{}
					mimetype, _, _ := mimes.FindMimeType(path, "text/plain")
					var rqstr io.Reader = nil
					var buf *iorw.Buffer = nil
					if mimetype == "application/json" {
						if len(prms) > 0 {
							buf = iorw.NewBuffer()
							enc := json.NewEncoder(buf)
							enc.Encode(prms)
							enc = nil
							rqstr = buf.Reader()
						}
					}
					remoteHeaders["Content-Type"] = mimetype
					if strings.HasSuffix(rscngepnt.path, "/") {
						if strings.HasPrefix(path, "/") {
							path = path[1:]
						}
					} else {
						if !strings.HasPrefix(path, "/") {
							path = "/" + path
						}
					}
					func() {
						if r, rerr := web.DefaultClient.Send(rscngepnt.schema+"://"+strings.Replace(rscngepnt.host+rscngepnt.path+path, "//", "/", -1), remoteHeaders, nil, rqstr); rerr == nil {
							modified = time.Now()
							if rc, _ := r.(io.ReadCloser); rc != nil {
								rs = rc
							} else {
								rs = iorw.NewEOFCloseSeekReader(r)
							}
						} else if rerr != nil {
							err = rerr
						}
						if buf != nil {
							buf.Close()
							buf = nil
						}
					}()
				}
			}
		}()
	}
	return
}

// RemoveResource - remove inline resource - true if found and removed and false if not exists
func (rscngepnt *ResourcingEndpoint) RemoveResource(path string) (rmvd bool) {
	if path != "" {
		if rs, rsok := rscngepnt.embeddedResources[path]; rsok {
			rmvd = rsok
			rs.Close()
		}
	}
	return
}

// Resource - return mapped resource interface{} by path
func (rscngepnt *ResourcingEndpoint) Resource(path string) (rs interface{}) {
	if path != "" {
		rs = rscngepnt.embeddedResources[path]
	}
	return
}

func (rscngepnt *ResourcingEndpoint) LastModified() (modified time.Time) {
	if rscngepnt != nil {
		if rscngepnt.isLocal {
			if path := rscngepnt.path; path != "" {
				if fi, fe := os.Stat(rscngepnt.path[:len(rscngepnt.path)-1]); fe == nil {
					modified = fi.ModTime()
				}
			}
		}
	}
	return
}

func nextResourcingEndpoint(rsngmngr *ResourcingManager, path string, raw bool, active bool, a ...interface{}) (rsngepnt *ResourcingEndpoint, rsngepntpath string) {
	isLocal := false
	if rsngepntpath = path; rsngepntpath == "./" {
		exepath := rsngepntpath // os.Args[0]
		if fis, _ := fsutils.FIND(exepath); len(fis) > 0 && !fis[0].IsDir() {
			rsngepntpath = strings.Replace(fis[0].AbsolutePath(), "\\", "/", -1)
			rsngepntpath = rsngepntpath[:strings.LastIndex(rsngepntpath, "/")+1]
		} else if len(fis) > 0 && fis[0].IsDir() {
			rsngepntpath = strings.Replace(fis[0].AbsolutePath(), "\\", "/", -1)
			rsngepntpath = rsngepntpath[:strings.LastIndex(rsngepntpath, "/")+1]
		} else {
			path = strings.Replace(exepath, "\\", "/", -1)
			path = path[:strings.LastIndex(path, "/")+1]
			rsngepntpath = path
		}
		isLocal = true
	}
	var fs FS = nil
	var al = len(a)
	if al > 0 {
		ai := 0
		for ai < al {
			if d := a[ai]; d != nil {
				if fsd, _ := d.(FS); fsd != nil {
					if fs == nil {
						fs = fsd
					}
					a = append(a[:ai], a[ai+1:])
					continue
				}
			}
			ai++
		}
	}
	if rsngepntpath != "" {
		rsngepntpath = strings.Replace(strings.TrimSpace(rsngepntpath), "\\", "/", -1)
		if strings.HasPrefix(rsngepntpath, "http://") || strings.HasPrefix(rsngepntpath, "https://") {
			_, err := url.ParseRequestURI(rsngepntpath)
			if err == nil {
				u, err := url.Parse(rsngepntpath)
				if err == nil && u.Scheme != "" && u.Host != "" {
					var querystring = ""
					if u.RawQuery == "" {
						querystring = ""
					} else {
						querystring = u.RawQuery
					}
					path = u.Path
					rsngepnt = &ResourcingEndpoint{rsngmngr: rsngmngr, isLocal: false, isRemote: true, embeddedResources: map[string]*EmbeddedResource{}, host: u.Host, schema: u.Scheme, querystring: querystring, path: path, isRaw: raw, isActive: active}
				}
			}
		} else {
			if fs == nil {
				if fi, fierr := os.Stat(rsngepntpath); fierr == nil {
					if fi.IsDir() {
						if rsngepntpath != "/" && rune(rsngepntpath[len(rsngepntpath)-1]) != '/' {
							rsngepntpath = rsngepntpath + "/"
						}
						rsngepnt = &ResourcingEndpoint{rsngmngr: rsngmngr, isLocal: true, cachableExtsBuffs: iocaching.NewBufferCache(), isRemote: false, isEmbedded: false, embeddedResources: map[string]*EmbeddedResource{}, host: "", schema: "", querystring: "", path: rsngepntpath, isRaw: raw, isActive: active, fs: nil}
					} else if pthzip, _ := lkpzpextindex(rsngepntpath); pthzip > -1 {
						rsngepnt = &ResourcingEndpoint{rsngmngr: rsngmngr, isLocal: true, cachableExtsBuffs: iocaching.NewBufferCache(), isRemote: false, isEmbedded: false, embeddedResources: map[string]*EmbeddedResource{}, host: "", schema: "", querystring: "", path: rsngepntpath, isRaw: raw, isActive: active, fs: nil}
					}
				} else if pthzip, pthzipext := lkpzpextindex(rsngepntpath); pthzip > -1 {
					if fi, fierr := os.Stat((rsngepntpath[:pthzip+len(pthzipext)])); fierr == nil {
						if !fi.IsDir() {
							rsngepnt = &ResourcingEndpoint{rsngmngr: rsngmngr, isLocal: true, cachableExtsBuffs: iocaching.NewBufferCache(), isRemote: false, isEmbedded: false, embeddedResources: map[string]*EmbeddedResource{}, host: "", schema: "", querystring: "", path: rsngepntpath, isRaw: raw, isActive: active, fs: nil}
						}
					}
				}
			} else {
				if isLocal {
					if fi, fierr := os.Stat(rsngepntpath); fierr == nil {
						if fi.IsDir() {
							if rsngepntpath != "/" && rune(rsngepntpath[len(rsngepntpath)-1]) != '/' {
								rsngepntpath = rsngepntpath + "/"
							}
							rsngepnt = &ResourcingEndpoint{rsngmngr: rsngmngr, isLocal: true, cachableExtsBuffs: iocaching.NewBufferCache(), isRemote: false, isEmbedded: false, embeddedResources: map[string]*EmbeddedResource{}, host: "", schema: "", querystring: "", path: rsngepntpath, isRaw: raw, isActive: active, fs: nil}
						}
					}
				} else {
					rsngepnt = &ResourcingEndpoint{rsngmngr: rsngmngr, isLocal: isLocal, cachableExtsBuffs: cachableExtsBuffs, isRemote: false, isEmbedded: true, embeddedResources: map[string]*EmbeddedResource{}, host: "", schema: "", querystring: "", path: "", isRaw: raw, isActive: active, fs: fs}
				}
			}
		}
	} else {
		rsngepnt = &ResourcingEndpoint{rsngmngr: rsngmngr, isLocal: false, isRemote: false, isEmbedded: true, embeddedResources: map[string]*EmbeddedResource{}, host: "", schema: "", querystring: "", path: "", isRaw: raw, isActive: active, fs: fs}
	}
	return
}

var cachableExts = map[string]bool{".sql": true, ".html": true, ".htm": true, ".svg": true, ".xml": true, ".js": true, ".css": true}

var cachableExtsBuffs = iocaching.NewBufferCache()
