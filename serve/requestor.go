package serve

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lnksnk/lnksnk/concurrent"
	"github.com/lnksnk/lnksnk/database"
	"github.com/lnksnk/lnksnk/ws"

	//"github.com/lnksnk/lnksnk/email/emailing"
	//"github.com/lnksnk/lnksnk/emailservice"
	//"github.com/lnksnk/lnksnk/emailservice/emailserve"
	"github.com/lnksnk/lnksnk/fsutils"
	"github.com/lnksnk/lnksnk/iorw"
	"github.com/lnksnk/lnksnk/iorw/active"
	"github.com/lnksnk/lnksnk/iorw/parsing"
	_ "github.com/lnksnk/lnksnk/iorw/parsing/minify"
	"github.com/lnksnk/lnksnk/parameters"
	"github.com/lnksnk/lnksnk/resources"
	"github.com/lnksnk/lnksnk/scheduling"
	"github.com/lnksnk/lnksnk/serve/serveio"
	"github.com/lnksnk/lnksnk/stdio/command"
)

var lastserial int64 = time.Now().UnixNano()

func nextserial() (nxsrl int64) {
	for {
		if nxsrl = time.Now().UnixNano(); atomic.CompareAndSwapInt64(&lastserial, atomic.LoadInt64(&lastserial), nxsrl) {
			break
		}
		time.Sleep(1 * time.Nanosecond)
	}
	return
}

func ProcessRequesterConn(conn net.Conn, activemap map[string]interface{}) {
	if conn != nil {
		if rqst, rqsterr := http.ReadRequest(bufio.NewReaderSize(conn, 65536)); rqsterr != nil {
			conn.Close()
			return
		} else if rqst != nil {
			rspns := NewResponseWriter(rqst, conn)
			defer rspns.Close()
			ProcessRequest("", rqst, rspns, activemap)
		}
	}
}

func ProcessIORequest(path string, stdout io.WriteCloser, stdin io.ReadCloser, activemap map[string]interface{}) {
	if stdin == nil {
		buf := iorw.NewBuffer()
		if path == "" {
			path = "/"
		}
		buf.Println("GET " + strings.Replace(path, "\\", "/", -1) + " HTTP/1.1")
		buf.Println()
		stdin = buf.Reader(true)
	}
	if rqst, rqsterr := http.ReadRequest(bufio.NewReaderSize(stdin, 65536)); rqsterr != nil {
		stdin.Close()
		return
	} else if rqst != nil {
		ProcessRequest(path, rqst, NewResponseWriter(rqst, stdout), activemap)
	}
}

func ServeHTTPRequest(w http.ResponseWriter, r *http.Request) {
	ProcessRequest(r.URL.Path, r, w, nil)
}

func ProcessRequestPath(path string, activemap map[string]interface{}, a ...interface{}) (err error) {
	var fs *fsutils.FSUtils = nil
	ai, al := 0, len(a)
	for ai < al {
		if fsd, _ := a[ai].(*fsutils.FSUtils); fsd != nil {
			if fs == nil {
				fs = fsd
			}
			a = append(a[:ai], a[ai+1:]...)
			al--
			continue
		}
		ai++
	}
	if fs == nil {
		fs = gblfs
	}
	err = internalRequest(path, nil, nil, fs, activemap)
	//err = internalServeRequest(path, nil, nil, fs, activemap)
	return
}

func ProcessRequest(path string, httprqst *http.Request, httprspns http.ResponseWriter, activemap map[string]interface{}, a ...interface{}) (err error) {
	var fs *fsutils.FSUtils = nil
	ai, al := 0, len(a)
	for ai < al {
		if fsd, _ := a[ai].(*fsutils.FSUtils); fsd != nil {
			if fs == nil {
				fs = fsd
			}
			a = append(a[:ai], a[ai+1:]...)
			al--
			continue
		}
		ai++
	}
	if fs == nil {
		fs = gblfs
	}
	if httprqst != nil && httprspns != nil {
		if ws, wserr := ws.NewServerReaderWriter(httprspns, httprqst); wserr == nil && ws != nil {

			return
		}
		prcsrdr := serveio.NewReader(httprqst)
		defer prcsrdr.Close()
		prcswtr := serveio.NewWriter(httprspns)
		defer prcswtr.Close()
		err = internalRequest(path, prcsrdr, prcswtr, fs, activemap, a...)
	}
	return
}

var gblfs = resources.GLOBALRSNG().FS()

func ParseEval(evalcode func(a ...interface{}) (val interface{}, err error), path, pathext string, pathmodified time.Time, Out io.Writer, In io.Reader, fs *fsutils.FSUtils, invertactive bool, fi fsutils.FileInfo, fnmodified func(modified time.Time), fnactiveraw func(rsraw bool, rsactive bool)) (err error) {
	if fi != nil && path != "" && path[0:1] != "/" {
		path += fi.Path()
	}
	err = parsing.Parse(pathmodified, path, pathext, Out, func() (f io.Reader, ferr error) {
		if fi != nil {
			f, ferr = fi.Open(fnactiveraw, fnmodified)
			return
		}
		return In, nil
	}, fs, invertactive, evalcode)
	return
}

func InvokeVM(a ...interface{}) (nvm *active.VM) {
	select {
	case nvm = <-chngvm:
	default:
		nvm = active.NewVM()
	}
	var terminal *terminals = nil
	var Out serveio.Writer = nil
	var In serveio.Reader = nil
	var params parameters.Parameters = nil
	var caching *concurrent.Map
	var invkcachng func() *concurrent.Map
	var activemap map[string]interface{} = nil
	var dbhnlr *database.DBMSHandler = nil

	//var emailsvchndl *emailservice.EMAILSvcHandler = nil
	var fi fsutils.FileInfo
	var fs *fsutils.FSUtils = nil
	ai, al := 0, len(a)
	for ai < al {
		if terminalfuncd, _ := a[ai].(func() *terminals); terminalfuncd != nil {
			if terminal == nil {
				terminal = terminalfuncd()
			}
			a = append(a[:ai], a[ai+1:]...)
			al--
			continue
		}
		if terminald, _ := a[ai].(*terminals); terminald != nil {
			if terminal == nil {
				terminal = terminald
			}
			a = append(a[:ai], a[ai+1:]...)
			al--
			continue
		}
		if Outd, _ := a[ai].(serveio.Writer); Outd != nil {
			if Out == nil {
				Out = Outd
			}
			a = append(a[:ai], a[ai+1:]...)
			al--
			continue
		}
		if Ind, _ := a[ai].(serveio.Reader); Ind != nil {
			if In == nil {
				In = Ind
			}
			a = append(a[:ai], a[ai+1:]...)
			al--
			continue
		}
		if paramsd, _ := a[ai].(parameters.Parameters); paramsd != nil {
			if params == nil {
				params = paramsd
			}
			a = append(a[:ai], a[ai+1:]...)
			al--
			continue
		}
		if invkcachingd, _ := a[ai].(func() *concurrent.Map); invkcachingd != nil {
			if invkcachng == nil {
				invkcachng = invkcachingd
			}
			a = append(a[:ai], a[ai+1:]...)
			al--
			continue
		}
		if activemapd, _ := a[ai].(map[string]interface{}); activemapd != nil {
			if activemap == nil {
				activemap = activemapd
			}
			a = append(a[:ai], a[ai+1:]...)
			al--
			continue
		}
		if dbhnlrd, _ := a[ai].(*database.DBMSHandler); dbhnlrd != nil {
			if dbhnlr == nil {
				dbhnlr = dbhnlrd
			}
			a = append(a[:ai], a[ai+1:]...)
			al--
			continue
		}
		/*if emailsvchnld, _ := a[ai].(*emailservice.EMAILSvcHandler); emailsvchnld != nil {
			if emailsvchndl == nil {
				emailsvchndl = emailsvchnld
			}
			a = append(a[:ai], a[ai+1:]...)
			al--
			continue
		}*/
		if fsd, _ := a[ai].(*fsutils.FSUtils); fsd != nil {
			if fs == nil {
				fs = fsd
			}
			a = append(a[:ai], a[ai+1:]...)
			al--
			continue
		}
		if fid, _ := a[ai].(fsutils.FileInfo); fid != nil {
			if fi == nil {
				fi = fid
			}
			a = append(a[:ai], a[ai+1:]...)
			al--
			continue
		}
		if fifuncd, _ := a[ai].(func() fsutils.FileInfo); fifuncd != nil {
			if fi == nil {
				fi = fifuncd()
			}
			a = append(a[:ai], a[ai+1:]...)
			al--
			continue
		}
		if a[ai] == nil {
			a = append(a[:ai], a[ai+1:]...)
			al--
			continue
		}
		ai++
	}
	nvm.ErrPrint = func(a ...interface{}) (vmerr error) {
		/*if Out != nil {
			Out.Print("<pre>ERR:\r\n")
			Out.Print(a...)
			Out.Print("\r\n</pre>")
		}*/
		return
	}

	nvm.FS = fs
	nvm.Set("fs", nvm.FS)
	nvm.Set("listen", LISTEN)
	nvm.Set("lstn", LISTEN)
	nvm.Set("terminal", terminal)
	nvm.Set("trm", terminal)
	nvm.Set("command", terminal)
	nvm.Set("cmd", terminal)
	nvm.Set("faf", func(rqstpath string, a ...interface{}) {
		go ProcessRequestPath(rqstpath, nil, a...)
	})
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
			if nvm.W == Out {
				nvm.SetPrinter(prsout)
				defer func() {
					nvm.SetPrinter(Out)
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
						prsevalerr = ParseEval(nvm.Eval, fitouse.Path(), fitouse.PathExt(), fitouse.ModTime(), prsout, nil, fstouse, invert, fitouse, nil, nil)
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
					prsevalerr = ParseEval(nvm.Eval, ":no-cache/"+suggestedroot, ".js", time.Now(), prsout, prsevalbuf.Clone(true).Reader(true), fstouse, invert, nil, nil, nil)
				}
			}()
		} else if prin != nil {
			prsevalerr = ParseEval(nvm.Eval, ":no-cache/"+suggestedroot, ".js", time.Now(), prsout, prin, fstouse, invert, nil, nil, nil)
		}
		return prsevalerr
	}

	nvm.Set("parseEval", fparseEval)

	nvm.Set("scheduling", SCHEDULING)
	nvm.Set("schdlng", SCHEDULING)
	//nvm.Set("caching", CHACHING)
	//nvm.Set("cchng", CHACHING)
	nvm.Set("db", dbhnlr)
	//nvm.Set("emailsvc", emailsvchndl)
	/*nvm.Set("email", EMAILING.ActiveEmailManager(nvm, func() parameters.Parameters {
		return params
	}, fs))*/
	for actvkey, actvval := range activemap {
		nvm.Set(actvkey, actvval)
	}

	nvm.Set("_params", map[string]interface{}{
		"set":       params.Set,
		"get":       params.Get,
		"type":      params.Type,
		"exist":     params.Exist,
		"fileExist": params.FileExist,
		"setFile":   params.SetFile,
		"getFile":   params.GetFile,
		"keys":      params.Keys,
		"fileKeys":  params.FileKeys,
	})
	nvm.Set("_cache", map[string]interface{}{
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
			if caching == nil && invkcachng != nil {
				caching = invkcachng()
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
	nvm.Set("_in", In)
	nvm.Set("_out", Out)
	nvm.R = In
	nvm.W = Out

	return nvm
}

/*func internalServeRequest(path string, In serveio.Reader, Out serveio.Writer, fs *fsutils.FSUtils, activemap map[string]interface{}, a ...interface{}) (err error) {
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

	var terminal *terminals = nil
	if terminal != nil {
		defer terminal.Close()
	}
	var fi fsutils.FileInfo = nil

	var vm *active.VM = nil
	var invokevm func() *active.VM

	var dbhnlr *database.DBMSHandler = DBMS.DBMSHandler(ctx, active.RuntimeFunc(func(functocall interface{}, args ...interface{}) interface{} {
		return invokevm().InvokeFunction(functocall, args...)
	}), params, fs, func(ina ...interface{}) (a []interface{}) {
		if len(ina) == 1 {
			if fia, _ := ina[0].(fsutils.FileInfo); fia != nil {
				dbvm := invokevm()
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
	defer dbhnlr.Dispose()

	invokevm = func() *active.VM {
		if vm != nil {
			return vm
		}
		vm = InvokeVM(vm, func() *terminals {
			if terminal == nil {
				terminal = newTerminal()
			}
			return terminal
		}, dbhnlr, params, Out, In, activemap, func() fsutils.FileInfo {
			return fi
		}, fs, invokecaching)
		return vm
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
	if vm != nil {
		defer func() {
			chngvm <- vm
		}()
	}

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

	var invertactive = false
	if strings.Contains(path, "/active:") {
		for strings.Contains(path, "/active:") {
			prepath := path[:strings.Index(path, "/active:")+1]
			path = prepath + path[strings.Index(path, "/active:")+len("/active:"):]
		}
		invertactive = true
	}
	if pathext != "" {
		if fis := fs.LS(path); len(fis) == 1 {
			mimetipe, istexttype, ismedia = mimes.FindMimeType(pathext, "text/plain")
			fi = fis[0]
			fnactiveraw(fi.IsRaw(), fi.IsActive())
			fnmodified(fi.ModTime())
		}
	}
	if fndapi, dbapierr := dbserve.ServeRequest("/db:", Out, In, path, dbhnlr, params, fs); fndapi || dbapierr != nil {
		return
	}
	if fi == nil && pathext == "" && strings.HasSuffix(path, "/") {
		for _, psblexts := range []string{".html", ".js"} {
			isactive = true
			if fis := fs.LS(path + "index" + psblexts); len(fis) == 1 {
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

	if istexttype || strings.Contains(mimetipe, "text/plain") {
		mimetipe += "; charset=utf-8"
	}
	if Out != nil {
		Out.Header().Set("Content-Type", mimetipe)
	}
	if fi != nil {
		if pathext != "" {

			if invertactive {
				if !israw && !ismedia {
					invertactive = true
					if !isactive {
						isactive = true
					}
				}
			}

			if !isactive && convertactive {
				isactive = true
			}

			if !israw && isactive {
				err = ParseEval(invokevm().Eval, path, pathext, pathmodified, Out, nil, fs, invertactive, fi, fnmodified, fnactiveraw)
			} else if israw || ismedia {
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
				} else {
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
				}
			} else {
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
			}
		}
	}
	return
}*/

type terminals struct {
	cmdprscs    *sync.Map
	cmdprscrefs *sync.Map
}

func newTerminal() (terms *terminals) {
	terms = &terminals{cmdprscs: &sync.Map{}, cmdprscrefs: &sync.Map{}}
	return
}

func (terms *terminals) Command(alias string, execargs ...string) (cmd *command.Command, err error) { // (cmd *osprc.Command, err error) {
	if terms != nil && alias != "" {
		execpath := ""
		if len(execargs) > 0 {
			execpath = execargs[0]
			execargs = execargs[1:]
		}
		if execpath != "" {
			if cmd, err = command.NewCommand(execpath, os.Environ(), execargs...); err == nil && cmd != nil {
				if terms.cmdprscs == nil {
					terms.cmdprscs = &sync.Map{}
				}
				if terms.cmdprscrefs != nil {
					if cmpiv, cmpivok := terms.cmdprscrefs.Load(alias); cmpivok {
						cmpi, _ := cmpiv.(int)
						if cmv, cmvok := terms.cmdprscs.Load(cmpi); cmvok {
							if cmpref, _ := cmv.(*command.Command); cmpref != nil {
								cmpref.Close()
							}
						}
					}
				} else {
					terms.cmdprscrefs = &sync.Map{}
				}
				terms.cmdprscs.Store(cmd.Pid, cmd)
				cmd.OnClose = func(prcid int) {
					if cmpiv, cmpivok := terms.cmdprscrefs.Load(alias); cmpivok {
						if cmpi, _ := cmpiv.(int); cmpi == prcid {
							terms.cmdprscrefs.Delete(alias)
						}
						if _, cmpvok := terms.cmdprscs.Load(prcid); cmpvok {
							terms.cmdprscs.Delete(prcid)
						}
					}
				}
			} else {
				if terms.cmdprscrefs != nil {
					if cmpiv, cmpivok := terms.cmdprscrefs.Load(alias); cmpivok {
						cmpi, _ := cmpiv.(int)
						if cmpv, cmpvok := terms.cmdprscs.Load(cmpi); cmpvok {
							cmd, _ = cmpv.(*command.Command)
						}
					}
				}
			}
		}
	}
	return
}

func (terms *terminals) Close() {
	if terms != nil {
		if cmdprscs := terms.cmdprscs; cmdprscs != nil {
			cmdprscs.Range(func(key, value any) bool {
				if cmd, _ := value.(*command.Command); cmd != nil {
					cmd.Close()
				}
				return true
			})
			terms.cmdprscs = nil
			if terms.cmdprscrefs != nil {
				terms.cmdprscrefs = nil
			}
		}
		if terms.cmdprscrefs != nil {
			terms.cmdprscrefs = nil
		}
	}
}

var chngvm = make(chan *active.VM)
var DBMS = database.GLOBALDBMS()
var SCHEDULING = scheduling.GLOBALSCHEDULING()
var CACHING = concurrent.NewMap()

type ListenApi interface {
	Serve(network string, addr string, tlsconf ...*tls.Config)
	ServeTLS(network string, addr string, orgname string, tlsconf ...*tls.Config)
	Shutdown(...string)
}

var LISTEN ListenApi = nil

func init() {
	active.DefaultParseFileInfo = parsing.ParseFileInfo
	go func() {
		for vm := range chngvm {
			go vm.Close()
		}
	}()
}
