package serve

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/lnksnk/lnksnk/concurrent"
	"github.com/lnksnk/lnksnk/database"
	"github.com/lnksnk/lnksnk/fsutils"
	"github.com/lnksnk/lnksnk/iorw"
	"github.com/lnksnk/lnksnk/iorw/active"
	"github.com/lnksnk/lnksnk/ja"
	"github.com/lnksnk/lnksnk/mimes"
	"github.com/lnksnk/lnksnk/parameters"
	"github.com/lnksnk/lnksnk/serve/serveio"
)

type internalWork struct {
	path      string
	ctxcnl    context.CancelCauseFunc
	In        serveio.Reader
	Out       serveio.Writer
	fs        *fsutils.FSUtils
	activemap map[string]interface{}
	a         []interface{}
	ctx       context.Context
}

func initInternWork(path string, In serveio.Reader, Out serveio.Writer, fs *fsutils.FSUtils, activemap map[string]interface{}, a ...interface{}) (intrnwrk *internalWork) {
	ctx, ctxcnl := context.WithCancelCause(context.Background())
	intrnwrk = &internalWork{ctx: ctx, ctxcnl: ctxcnl, path: path, In: In, Out: Out, fs: fs, activemap: activemap}
	intrnwrk.a = append(intrnwrk.a, a...)
	return
}

func internalRequest(path string, In serveio.Reader, Out serveio.Writer, fs *fsutils.FSUtils, activemap map[string]interface{}, a ...interface{}) (err error) {
	ctx, ctxcnxl := context.WithCancelCause(context.Background())
	go func() {
		ctxcnxl(_internalRequest(path, In, Out, fs, activemap, a...))
	}()
	<-ctx.Done()
	if ctxerr := ctx.Err(); ctxerr != context.Canceled {
		err = ctxerr
	}
	return
}

func (intrnwk *internalWork) close() {
	if intrnwk != nil {
		intrnwk.a = nil
		intrnwk.activemap = nil
		intrnwk.In = nil
		intrnwk.Out = nil
		intrnwk.ctx = nil
		intrnwk.ctxcnl = nil
		intrnwk.fs = nil
	}
}

func (intrnwk *internalWork) do() {
	intrnwk.ctxcnl(_internalRequest(intrnwk.path, intrnwk.In, intrnwk.Out, intrnwk.fs, intrnwk.activemap, intrnwk.a...))
}

func _internalRequest(path string, In serveio.Reader, Out serveio.Writer, fs *fsutils.FSUtils, activemap map[string]interface{}, a ...interface{}) (err error) {
	var caching *concurrent.Map
	var invokecaching = func() *concurrent.Map {
		if caching == nil {
			caching = concurrent.NewMap()
		}
		return caching
	}
	defer func() {
		if caching != nil {
			go caching.Dispose()
		}
	}()
	params := parameters.NewParameters()
	defer params.CleanupParameters()
	var ctx context.Context = nil

	if ctx != nil {
		ctx = nil
	}
	if In != nil {
		ctx = In.Context()
		defer In.Close()
		parameters.LoadParametersFromHTTPRequest(params, In.HttpR())
		if path == "" {
			path = In.Path()
		}
	} else {
		if path != "" {
			path = strings.Replace(path, "\\", "/", -1)
		}
	}
	if strings.Contains(path, "?") {
		parameters.LoadParametersFromRawURL(params, path)
	}

	if Out != nil {
		defer Out.Close()
	}

	var rangeOffset = func() int64 {
		if In != nil {
			return In.RangeOffset()
		}
		return 0
	}()
	var rangeType = func() string {
		if In != nil {
			return In.RangeType()
		}
		return ""
	}()

	var pathext = filepath.Ext(path)
	var pathmodified time.Time = time.Now()

	var fnmodified = func(modified time.Time) {
		pathmodified = modified
	}
	var israw = false
	var convertactive = false

	var mimetipe, istexttype, ismedia = mimes.FindMimeType(pathext, "text/plain")
	var isactive = istexttype

	var fnactiveraw = func(rsraw bool, rsactive bool) {
		if israw = rsraw; !israw {
			if isactive {
				if !convertactive {
					convertactive = rsactive
				}
			}
		} else {
			isactive = false
		}
	}

	var terminal *terminals = nil
	defer func() {
		if terminal != nil {
			terminal.Close()
		}
	}()

	var invertactive = false
	if strings.Contains(path, "/active:") {
		for strings.Contains(path, "/active:") {
			prepath := path[:strings.Index(path, "/active:")+1]
			path = prepath + path[strings.Index(path, "/active:")+len("/active:"):]
		}
		invertactive = true
	}

	var fi fsutils.FileInfo
	if pathext != "" {
		if fis := fs.LS(path); len(fis) == 1 {
			mimetipe, istexttype, ismedia = mimes.FindMimeType(pathext, "text/plain")
			fi = fis[0]
			fnactiveraw(fi.IsRaw(), fi.IsActive())
			fnmodified(fi.ModTime())
		}
	} else {
		if fis := fs.LS(path); len(fis) == 1 {
			mimetipe, istexttype, ismedia = mimes.FindMimeType(pathext, "text/plain")
			fi = nil
			if fis[0].IsDir() {
				for _, psblexts := range []string{".html", ".js", ".json"} {
					isactive = true
					if fis := fs.LS(fis[0].Path() + "index" + psblexts); len(fis) == 1 {
						fi = fis[0]
						path = fi.Path()
						mimetipe, istexttype, ismedia = mimes.FindMimeType(psblexts, "text/plain")
						pathext = fi.PathExt()

						fnactiveraw(fi.IsRaw(), fi.IsActive())
						fnmodified(fi.ModTime())
						break
					}
				}
			}
		}
	}

	if istexttype || strings.Contains(mimetipe, "text/plain") {
		mimetipe += "; charset=utf-8"
	}
	if Out != nil {
		Out.Header().Set("Content-Type", mimetipe)
	}
	if fi == nil {
		return
	}

	if !isactive && convertactive {
		isactive = true
	}

	if isactive {
		vm := active.NewVM()
		vm.Set("_params", map[string]interface{}{
			"get":   params.Parameter,
			"set":   params.SetParameter,
			"exist": params.ContainsParameter,
		})
		vm.Set("fs", fs)
		vm.FS = fs
		var fparseEval = func(prsout io.Writer, evalrt interface{}, a ...interface{}) (prsevalerr error) {
			var invert bool = false
			var fitouse fsutils.FileInfo = nil
			var fstouse *fsutils.FSUtils = nil
			var prin, _ = evalrt.(io.Reader)
			var evalroot, _ = evalrt.(string)
			var suggestedroot = "/"
			if prsout == nil {
				prsout = Out
			} else if prsout != Out {
				if vm.W == Out {
					vm.SetPrinter(prsout)
					defer func() {
						vm.SetPrinter(Out)
					}()
				}
			}
			if len(a) > 0 {
				if inv, invok := a[0].(bool); invok {
					invert = inv
					a = a[1:]
				}
			}
			ai := 0
			al := len(a)
			for ai < al {
				d := a[ai]
				if fid, _ := d.(fsutils.FileInfo); fid != nil {
					if fitouse == nil {
						fitouse = fid
					}
					a = append(a[:ai], a[ai+1:])
					al--
					continue
				}
				if fsd, _ := d.(*fsutils.FSUtils); fsd != nil {
					if fstouse == nil {
						fstouse = fsd
					}
					a = append(a[:ai], a[ai+1:])
					al--
					continue
				}
				ai++
			}

			if fstouse == nil && fs != nil {
				fstouse = fs
			}

			if fstouse != nil {
				if fitouse == nil {
					if evalroot != "" && prin == nil {
						if fios := fs.LS(evalroot); len(fios) == 1 {
							fitouse = fios[0]
							prsevalerr = ParseEval(vm.Eval, fitouse.Path(), fitouse.PathExt(), fitouse.ModTime(), prsout, nil, fstouse, invert, fitouse, nil, nil)
							return
						}
					}
					fitouse = fi
				}
			}

			if fitouse != nil {
				suggestedroot = fitouse.PathRoot()
			}

			if evalroot != "" && prin == nil {
				prin = strings.NewReader(evalroot)
			}

			if prin == nil && len(a) > 0 {
				func() {
					var prsevalbuf = iorw.NewBuffer()
					defer prsevalbuf.Clear()
					prsevalbuf.Print(a...)
					if prsevalbuf.Size() > 0 {
						prsevalerr = ParseEval(vm.Eval, ":no-cache/"+suggestedroot, ".js", time.Now(), prsout, prsevalbuf.Clone(true).Reader(true), fstouse, invert, nil, nil, nil)
					}
				}()
			} else if prin != nil {
				prsevalerr = ParseEval(vm.Eval, ":no-cache/"+suggestedroot, ".js", time.Now(), prsout, prin, fstouse, invert, nil, nil, nil)
			}
			return prsevalerr
		}
		vm.Set("parseEval", fparseEval)
		vm.Set("trm", func() *terminals {
			if terminal == nil {
				terminal = newTerminal()
			}
			return terminal
		})
		vm.Set("listen", LISTEN)
		var dbhnlr *database.DBMSHandler = DBMS.DBMSHandler(ctx, vm, params, fs, func(ina ...interface{}) (a []interface{}) {
			if len(ina) == 1 {
				if fia, _ := ina[0].(fsutils.FileInfo); fia != nil {
					dbvm := vm
					stmntoutbuf := iorw.NewBuffer()
					defer stmntoutbuf.Close()
					vmw := dbvm.W
					vm.W = stmntoutbuf
					if evalerr := ParseEval(dbvm.Eval, fia.Path(), fia.PathExt(), fia.ModTime(), stmntoutbuf, nil, fs, false, fia, nil, nil); evalerr == nil {
						a = append(a, stmntoutbuf.Clone(true).Reader(true))
					}
					dbvm.W = vmw
				}
			} else {
				a = append(a, ina...)
			}
			return
		})
		vm.Set("db", dbhnlr)
		defer func() {
			if vm != nil {
				go vm.Close()
				vm = nil
			}
			if dbhnlr != nil {
				go dbhnlr.Dispose()
				dbhnlr = nil
			}
		}()
		vm.Set("_cache", map[string]interface{}{
			"count": func() (cnt int) {
				if caching != nil {
					return caching.Count()
				}
				return
			},
			"del": func(keys ...interface{}) {
				if caching != nil {
					caching.Del(keys...)
				}
			},
			"find": func(k ...interface{}) (value interface{}, found bool) {
				if caching != nil {
					value, found = caching.Find(k...)
				}
				return
			},
			"exist": func(key interface{}) (exist bool) {
				if caching != nil {
					exist = caching.Exist(key)
				}
				return
			},
			"get": func(key interface{}) (value interface{}, loaded bool) {
				if caching != nil {
					value, loaded = caching.Get(key)
				}
				return
			},
			"set": func(key, value interface{}) {
				if caching == nil && invokecaching != nil {
					caching = invokecaching()
					caching.Set(key, value)
				}
				if caching != nil {
					caching.Set(key, value)
				}
			},
			"forEach": func(ietrfunc func(key any, value any) bool) {
				if ietrfunc != nil {
					if caching != nil && ietrfunc != nil {
						caching.ForEach(func(key, value any, first, last bool) bool {
							return !ietrfunc(key, value)
						})
					}
				}
			},
			"keys": func() (keys []interface{}) {
				if caching != nil {
					return caching.Keys()
				}
				return
			},
		})
		vm.Set("_in", In)
		vm.Set("_out", Out)
		vm.R = In
		vm.W = Out
		err = ParseEval(vm.Eval, path, pathext, pathmodified, Out, nil, fs, invertactive, fi, fnmodified, fnactiveraw)
		return
	}
	if ismedia {
		if f, ferr := fi.Open(); ferr == nil {
			defer f.Close()
			if rssize := fi.Size(); rssize > 0 {
				var eofrs *iorw.EOFCloseSeekReader = nil
				if eofrs, _ = f.(*iorw.EOFCloseSeekReader); eofrs == nil {
					eofrs = iorw.NewEOFCloseSeekReader(f, false)
				}
				if eofrs != nil {
					if rangeOffset == -1 {
						rangeOffset = 0
					} else {
						eofrs.Seek(rangeOffset, 0)
					}
					if rssize > 0 {
						if rangeType == "bytes" && rangeOffset > -1 {
							maxoffset := int64(0)
							maxlen := int64(0)
							if maxoffset = rangeOffset + (rssize - rangeOffset); maxoffset > 0 {
								maxlen = maxoffset - rangeOffset
								maxoffset--
							}

							if maxoffset < rangeOffset {
								maxoffset = rangeOffset
								maxlen = 0
							}

							if maxlen > 1024*1024 {
								maxlen = 1024 * 1024
								maxoffset = rangeOffset + (maxlen - 1)
							}
							contentrange := fmt.Sprintf("%s %d-%d/%d", In.RangeType(), rangeOffset, maxoffset, rssize)
							if Out != nil {
								//Out.Header().Set("Accept-Ranges", "bytes")
								Out.Header().Set("Content-Range", contentrange)
								Out.Header().Set("Content-Length", fmt.Sprintf("%d", maxlen))
							}
							eofrs.SetMaxRead(maxlen)
							Out.MaxWriteSize(maxlen)
							if Out != nil {
								Out.WriteHeader(206)
							}
						} else {
							if Out != nil {
								Out.Header().Set("Accept-Ranges", "bytes")
								Out.Header().Set("Content-Length", fmt.Sprintf("%d", rssize))
							}
							eofrs.SetMaxRead(rssize)
							Out.MaxWriteSize(rssize)
						}
					}
					Out.BPrint(eofrs)
				}
			}
		}
		return
	}
	if israw {
		if Out != nil {
			if fi != nil {
				Out.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
				Out.WriteHeader(200)
				if f, ferr := fi.Open(); ferr == nil {
					if f != nil {
						defer f.Close()
						Out.BPrint(io.LimitReader(f, fi.Size()))
					}
				}
			}
		}
		return
	}
	if Out != nil {
		if f, ferr := fi.Open(); ferr == nil {
			if f != nil {
				defer f.Close()
				Out.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
				Out.WriteHeader(200)
				Out.BPrint(io.LimitReader(f, fi.Size()))
			}
		}
	}
	return
}

var testprgm *ja.Program = nil

func init() {
	testprgm, _ = ja.Compile("", `print("<html><body>"); print("<ul>"); var rec=db.qry("lnksnk_etl","select now() d;"); while(rec.next()) {
		print("<li>",rec.data()[0],"</li>");
	} print("</ul>"); print("</body></html>");`, false)
}