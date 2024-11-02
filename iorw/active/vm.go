package active

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/lnksnk/lnksnk/fsutils"
	"github.com/lnksnk/lnksnk/iorw"

	"github.com/lnksnk/lnksnk/ja"
	"github.com/lnksnk/lnksnk/ja/parser"
)

type VM struct {
	vm            *ja.Runtime
	objmap        map[string]interface{}
	DisposeObject func(string, interface{})
	W             io.Writer
	R             io.Reader
	buffs         map[*iorw.Buffer]*iorw.Buffer
	ErrPrint      func(...interface{}) error
	FS            *fsutils.FSUtils
	ImportModule  func(referencingScriptOrModule interface{}, specifier ja.Value, promiseCapability interface{})
}

func NewVM(a ...interface{}) (vm *VM) {
	var w io.Writer = nil
	var r io.Reader = nil
	var stngs map[string]interface{} = nil
	for _, d := range a {
		if d != nil {
			if wd, _ := d.(io.Writer); wd != nil {
				if w == nil {
					w = wd
				}
			} else if rd, _ := d.(io.Reader); rd != nil {
				if r == nil {
					r = rd
				}
			} else if stngsd, _ := d.(map[string]interface{}); stngsd != nil {
				if stngs == nil {
					stngs = map[string]interface{}{}
					for stngk, stngv := range stngsd {
						stngs[stngk] = stngv
					}
				}
			}
		}
	}
	vm = &VM{vm: ja.New(), W: w, R: r, objmap: map[string]interface{}{}}
	vm.Set("console", map[string]interface{}{
		"log": func(a ...interface{}) {
			iorw.Fprintln(os.Stdout, a...)
		},
		"error": func(a ...interface{}) {
			iorw.Fprintln(os.Stdout, a...)
		},
		"warn": func(a ...interface{}) {
			iorw.Fprintln(os.Stdout, a...)
		},
	})
	vm.vm.RunProgram(adhocPrgm)
	var fldmppr = &fieldmapper{fldmppr: ja.UncapFieldNameMapper()}
	vm.vm.SetFieldNameMapper(fldmppr)
	for stngk, stngv := range stngs {
		if stngv != nil {
			if strings.EqualFold(stngk, "ERRPRINT") {
				if errprint, _ := stngv.(func(a ...interface{}) error); errprint != nil {
					if vm.ErrPrint == nil {
						vm.ErrPrint = errprint
					}
				}
			}
		}
		delete(stngs, stngk)
	}
	vm.vm.SetImportModule(func(modname string, namedimports ...[][]string) bool {
		return DefaultModuleManager.RunModule(vm.vm, vm.FS, modname, namedimports...) == nil
	})
	vm.vm.SetRequire(func(modname string) (exports *ja.Object) {
		exports, _ = DefaultModuleManager.Require(vm.vm, vm.FS, modname)
		return
	})

	vm.Set("include", func(modname string) bool {
		IncludeModule(vm.vm, modname)
		return true
	})
	vm.Set("setPrinter", vm.SetPrinter)
	vm.Set("print", vm.Print)
	vm.Set("println", vm.Println)
	vm.Set("binwrite", vm.Write)

	vm.Set("setReader", vm.SetReader)
	vm.Set("binread", vm.Read)
	vm.Set("readln", vm.Readln)
	vm.Set("readlines", vm.ReadLines)
	vm.Set("readAll", vm.ReadAll)
	vm.Set("sleep", func(d int64, a ...interface{}) interface{} {
		time.Sleep(time.Duration(d) * time.Millisecond)
		if al := len(a); al > 0 {
			if al == 1 {
				return vm.InvokeFunction(a[0])
			} else if al > 1 {
				return vm.InvokeFunction(a[0], a[1:]...)
			}
		}
		return nil
	})
	vm.Set("sleepnano", func(d int64, a ...interface{}) interface{} {
		time.Sleep(time.Duration(d) * time.Nanosecond)
		if al := len(a); al > 0 {
			if al == 1 {
				return vm.InvokeFunction(a[0])
			} else if al > 1 {
				return vm.InvokeFunction(a[0], a[1:]...)
			}
		}
		return nil
	})
	vm.Set("sleepsec", func(d int64, a ...interface{}) interface{} {
		time.Sleep(time.Duration(d) * time.Second)
		if al := len(a); al > 0 {
			if al == 1 {
				return vm.InvokeFunction(a[0])
			} else if al > 1 {
				return vm.InvokeFunction(a[0], a[1:]...)
			}
		}
		return nil
	})
	vm.Set("sleepmin", func(d int64, a ...interface{}) interface{} {
		time.Sleep(time.Duration(d) * time.Minute)
		if al := len(a); al > 0 {
			if al == 1 {
				return vm.InvokeFunction(a[0])
			} else if al > 1 {
				return vm.InvokeFunction(a[0], a[1:]...)
			}
		}
		return nil
	})
	vm.Set("buffer", func() (buf *iorw.Buffer) {
		buf = iorw.NewBuffer()
		if vm.buffs == nil {
			vm.buffs = map[*iorw.Buffer]*iorw.Buffer{}
		}
		buf.OnClose = func(b *iorw.Buffer) {
			delete(vm.buffs, b)
		}
		return
	})
	return
}

type fieldmapper struct {
	fldmppr ja.FieldNameMapper
}

// FieldName returns a JavaScript name for the given struct field in the given type.
// If this method returns "" the field becomes hidden.
func (fldmppr *fieldmapper) FieldName(t reflect.Type, f reflect.StructField) (fldnme string) {
	if f.Tag != "" {
		fldnme = f.Tag.Get("json")
	} else {
		fldnme = uncapitalize(f.Name) // fldmppr.fldmppr.FieldName(t, f)
	}
	return
}

// MethodName returns a JavaScript name for the given method in the given type.
// If this method returns "" the method becomes hidden.
func (fldmppr *fieldmapper) MethodName(t reflect.Type, m reflect.Method) (mthdnme string) {
	mthdnme = uncapitalize(m.Name)
	return
}

func uncapitalize(s string) (nme string) {
	if sl := len(s); sl > 0 {
		var nrxtsr = rune(0)
		for sn := range s {
			sr := s[sn]
			if 'A' <= sr && sr <= 'Z' {
				sr += 'a' - 'A'
				nme += string(sr)
			} else {
				nme += string(sr)
			}
			if sn <= (sl-1)-1 {
				nrxtsr = rune(s[sn+1])
			} else {
				nrxtsr = rune(0)
			}
			if 'a' <= nrxtsr && nrxtsr <= 'z' {
				nme += s[sn+1:]
				break
			}
		}
	}
	return nme
}

func (vm *VM) ImportModuleDynamically(referencingScriptOrModule interface{}, specifier ja.Value, promiseCapability interface{}) {

}

func (vm *VM) Get(objname string) (obj interface{}) {
	if vm != nil {
		if objmap := vm.objmap; objmap != nil {
			obj = objmap[objname]
		}
	}
	return
}

func (vm *VM) Set(objname string, obj interface{}) {
	if vm != nil && vm.objmap != nil && objname != "" {
		objm, objok := vm.objmap[objname]
		if objok && &objm != &obj {
			if objm != nil {
				vm.Remove(objname)
				if vm.vm != nil {
					vm.objmap[objname] = obj
					vm.vm.Set(objname, obj)
				}
			}
		} else {
			if vm.vm != nil {
				vm.objmap[objname] = obj
				vm.vm.Set(objname, obj)
			}
		}
	}
}

func (vm *VM) InvokeFunction(functocall interface{}, args ...interface{}) (result interface{}) {
	if functocall != nil {
		if vm != nil && vm.vm != nil {
			var fnccallargs []ja.Value = nil
			var argsn = 0

			for argsn < len(args) {
				if fnccallargs == nil {
					fnccallargs = make([]ja.Value, len(args))
				}
				fnccallargs[argsn] = vm.vm.ToValue(args[argsn])
				argsn++
			}
			if atvfunc, atvfuncok := functocall.(func(ja.FunctionCall) ja.Value); atvfuncok {
				if len(fnccallargs) == 0 || fnccallargs == nil {
					fnccallargs = []ja.Value{}
				}
				var funccll = ja.FunctionCall{This: ja.Undefined(), Arguments: fnccallargs}
				if rsltval := atvfunc(funccll); rsltval != nil {
					result = rsltval.Export()
				}
			}
		}
	}
	return
}

func (vm *VM) Remove(objname string) {
	if vm != nil && objname != "" {
		if vm.objmap != nil {
			if _, objok := vm.objmap[objname]; objok {
				if vm.vm != nil {
					if glblobj := vm.vm.GlobalObject(); glblobj != nil {
						glblobj.Delete(objname)
					}
				}
				vm.objmap[objname] = nil
				delete(vm.objmap, objname)
			}
		}
	}
}

func (vm *VM) SetReaderPrinter(r io.Reader, w io.Writer) {
	vm.SetReader(r)
	vm.SetPrinter(w)
}

func (vm *VM) SetReader(r io.Reader) {
	if vm != nil && vm.R != r {
		vm.R = r
	}
}

func (vm *VM) Read(p ...byte) (n int, err error) {
	if vm != nil && vm.R != nil {
		n, err = vm.R.Read(p)
	}
	return
}

func (vm *VM) Readln() (ln string, err error) {
	if vm != nil && vm.R != nil {
		ln, err = iorw.ReadLine(vm.R)
	}
	return
}

func (vm *VM) ReadLines() (lines []string, err error) {
	if vm != nil && vm.R != nil {
		lines, err = iorw.ReadLines(vm.R)
	}
	return
}

func (vm *VM) ReadAll() (all string, err error) {
	if vm != nil && vm.R != nil {
		all, err = iorw.ReaderToString(vm.R)
	}
	return
}

func (vm *VM) SetPrinter(w io.Writer) {
	if vm != nil && vm.W != w {
		vm.W = w
	}
}

func (vm *VM) Print(a ...interface{}) (err error) {
	if vm != nil && vm.W != nil {
		err = iorw.Fprint(vm.W, a...)
	}
	return
}

func (vm *VM) Println(a ...interface{}) (err error) {
	if vm != nil && vm.W != nil {
		err = iorw.Fprintln(vm.W, a...)
	}
	return
}

func (vm *VM) Write(p ...byte) (n int, err error) {
	if vm != nil && vm.W != nil {
		n, err = vm.W.Write(p)
	}
	return
}

var DefaultTransformCode func(code string) (transformedcode string, errors []string, warnings []string)

type programModElemManager struct {
	prgmodelms *sync.Map
}

func (prgmodmngr *programModElemManager) InvokeModule(fs *fsutils.FSUtils, specifier string) (invoked bool, invkerr error) {
	if prgmodmngr == nil {
		return
	}

	return
}

func (prgmodmngr *programModElemManager) Require(vm *ja.Runtime, fs *fsutils.FSUtils, specifier string) (export *ja.Object, err error) {
	if prgmodmngr == nil {
		return
	}
	if specifier == "" {
		err = fmt.Errorf("%s", "No specifier provided")
		if vm != nil {
			vm.Interrupt(err)
			vm.ClearInterrupt()
		}
		return
	}
	if fs != nil {
		prgmodelm := prgmodmngr.Module(specifier)
		if prgmodelm == nil {
			if prgmodelm, err = newProgramModElement(prgmodmngr, specifier, fs, nil); prgmodelm == nil || err != nil {
				err = errors.Join(err, fmt.Errorf("%s", "Unable to load "+specifier))
				if vm != nil {
					vm.Interrupt(err)
					vm.ClearInterrupt()
				}
				return
			}
			prgmodmngr.prgmodelms.Store(specifier, prgmodelm)
			export = prgmodelm.RequireMod(vm, fs)
			return
		}
	retry:
		if fs.EXIST(specifier) {
			fi := fs.LS(specifier)[0]
			if fi.IsDir() {
				specifier = fi.Path() + "index.js"
				fi = nil
				goto retry
			}
			if fi.ModTime() != prgmodelm.modfied {
				prgmodelms := prgmodmngr.prgmodelms
				if prgmodelms != nil {
					prgmodelms.Delete(specifier)
				}
				if prgmodelm, err = newProgramModElement(prgmodmngr, specifier, nil, fi); err != nil {
					return
				}
				prgmodelms.Store(specifier, prgmodelm)
				export = prgmodelm.RequireMod(vm, fs)
				return
			}
			export = prgmodelm.RequireMod(vm, fs)
			return
		}
	}
	return
}

func (prgmodmngr *programModElemManager) RunModule(vm *ja.Runtime, fs *fsutils.FSUtils, specifier string, namedimports ...[][]string) (err error) {
	if prgmodmngr == nil {
		return
	}
	if specifier == "" {
		err = fmt.Errorf("%s", "No specifier provided")
		if vm != nil {
			vm.Interrupt(err)
			vm.ClearInterrupt()
		}
		return
	}
	if fs != nil {
		prgmodelm := prgmodmngr.Module(specifier)
		if prgmodelm == nil {
			if prgmodelm, err = newProgramModElement(prgmodmngr, specifier, fs, nil); prgmodelm == nil || err != nil {
				err = errors.Join(err, fmt.Errorf("%s", "Unable to load "+specifier))
				if vm != nil {
					vm.Interrupt(err)
					vm.ClearInterrupt()
				}
				return
			}
			prgmodmngr.prgmodelms.Store(specifier, prgmodelm)
			prgmodelm.RunMod(vm, fs, namedimports...)
			return
		}
	retry:
		if fs.EXIST(specifier) {
			fi := fs.LS(specifier)[0]
			if fi.IsDir() {
				specifier = fi.Path() + "index.js"
				fi = nil
				goto retry
			}
			if fi.ModTime() != prgmodelm.modfied {
				prgmodelms := prgmodmngr.prgmodelms
				if prgmodelms != nil {
					prgmodelms.Delete(specifier)
				}
				if prgmodelm, err = newProgramModElement(prgmodmngr, specifier, nil, fi); err != nil {
					return
				}
				prgmodelms.Store(specifier, prgmodelm)
				prgmodelm.RunMod(vm, fs, namedimports...)
			}
			prgmodelm.RunMod(vm, fs, namedimports...)
			return
		}
	}
	return
}

var DefaultParseModeCode = func(prgmodmngr *programModElemManager, fi fsutils.FileInfo, fs *fsutils.FSUtils) (cde string, prserr error) {
	if DefaultParseFileInfo != nil {
		bfout := iorw.NewBuffer()
		prserr = DefaultParseFileInfo(fi, fs, ".js", bfout, true, func(a ...interface{}) (result interface{}, err error) {
			p, perr := Compile(a...)
			if perr != nil {
				prserr = perr
			}
			cde = p.Src()

			return
		})
	}
	return
}

var DefaultParseFileInfo func(fi fsutils.FileInfo, fs *fsutils.FSUtils, defaultext string, out io.Writer, invertActive bool, evalcode func(...interface{}) (interface{}, error), a ...interface{}) (prserr error)

func newProgramModElement(prgmodmngr *programModElemManager, specifier string, fs *fsutils.FSUtils, fi fsutils.FileInfo) (prgmodelm *programModElement, err error) {
	if specifier != "" {
		if fi == nil {
			if fs != nil {
				fis := fs.LS(specifier)
				if fisl := len(fis); fisl > 0 {
					if !fis[0].IsDir() {
						fi = fis[0]
						goto doit
					}
					if fis[0].IsDir() {
						retryspecifier := fis[0].Path() + "index.js"
						if fis = fs.LS(specifier); len(fis) == 1 {
							specifier = retryspecifier
							fi = fis[0]
							goto doit
						}
					}
				}
				return nil, fmt.Errorf("specifier %s is a director", specifier)
			}
		}
	doit:
		if fi != nil {
			src := ""
			if DefaultParseModeCode != nil {
				if src, err = DefaultParseModeCode(prgmodmngr, fi, fs); err != nil {
					return nil, err
				}
			}
			if src == "" {
				if fr, _ := fi.Open(); fr != nil {
					defer fr.Close()

					if src, _ = iorw.ReaderToString(fr); src == "" {
						return nil, fmt.Errorf("empty source for specifier %s", specifier)
					}
				}
			}
			p, perr := ja.ParseModule(specifier, src, func(referencingScriptOrModule interface{}, modspecifier string) (ja.ModuleRecord, error) {
				if prgmodelm = prgmodmngr.Module(modspecifier); prgmodelm != nil {
					return prgmodelm.m, nil
				}
				return nil, fmt.Errorf("unable to load specifier %s", modspecifier)
			})
			if perr != nil {
				err = perr
				return
			}
			if err = p.Link(); err != nil {
				p = nil
				return
			}
			prgmodelm = &programModElement{modfied: fi.ModTime(), m: p, prgmodmngr: prgmodmngr}
		}
	}
	return
}

func (prgmodmngr *programModElemManager) Module(specifier string) *programModElement {
	if prgmodmngr == nil || specifier == "" {
		return nil
	}
	if prgmodelms := prgmodmngr.prgmodelms; prgmodelms != nil {
		if modv, modok := prgmodelms.Load(specifier); modok {
			return modv.(*programModElement)
		}
	}
	return nil
}

var DefaultModuleManager = &programModElemManager{prgmodelms: &sync.Map{}}

type programModElement struct {
	prgmodmngr *programModElemManager
	modfied    time.Time
	m          ja.ModuleRecord
	prgm       *ja.Program
}

func (prgmodElm *programModElement) Invoke(fs *fsutils.FSUtils, specifier string) {

}

func (prgmodElm *programModElement) RunMod(vm *ja.Runtime, fs *fsutils.FSUtils, namedimports ...[][]string) {
	if prgmodElm == nil {
		return
	}

	if nmspce := prgmodElm.RequireMod(vm, fs); nmspce != nil {
		for _, nmdimprt := range namedimports {
			for _, imprtthis := range nmdimprt {
				if imprtthisl := len(imprtthis); imprtthisl > 0 {
					idntys := imprtthis[0]
					if idntys != "" {
						if imprtthisl > 1 {
							if aliass := imprtthis[1]; aliass != "" {
								vm.Set(aliass, nmspce.Get(idntys))
								continue
							}
						}
						vm.Set(idntys, nmspce.Get(idntys))
					}
				}
			}
		}
	}
}

func (prgmodElm *programModElement) RequireMod(vm *ja.Runtime, fs *fsutils.FSUtils) (exports *ja.Object) {
	if prgmodElm == nil {
		return
	}

	m := prgmodElm.m

	if vm != nil {
		evalprms := m.Evaluate(vm)
		if evalprms.State() == ja.PromiseStateFulfilled {
			exports = vm.NamespaceObjectFor(m)
		}
	}
	return
}

type parseerr struct {
	cde string
	err error
}

func (prserr *parseerr) Error() string {
	return prserr.err.Error()
}

func (prserr *parseerr) Code() string {
	return prserr.cde
}

func Compile(a ...interface{}) (p *ja.Program, perr error) {
	var ai, ail = 0, len(a)
	var cdes = ""
	var chdprgm *ja.Program = nil
	var setchdprgm func(interface{}, error, error)
	for ai < ail {
		if chdpgrmd, chdpgrmdok := a[ai].(*ja.Program); chdpgrmdok {
			if chdprgm == nil && chdpgrmd != nil {
				chdprgm = chdpgrmd
			}
			ail--
			a = append(a[:ai], a[ai+1:]...)
			continue
		}
		if setchdpgrmd, setchdpgrmdok := a[ai].(func(interface{}, error, error)); setchdpgrmdok {
			if setchdprgm == nil && setchdpgrmd != nil {
				setchdprgm = setchdpgrmd
			}
			ail--
			a = append(a[:ai], a[ai+1:]...)
			continue
		}
		ai++
	}
	if p = chdprgm; p == nil {
		if p == nil {
			var cde = iorw.NewMultiArgsReader(a...)
			defer cde.Close()
			cdes, _ = cde.ReadAll()

			prsd, prsderr := parser.ParseFile(nil, "", cdes, 0, parser.WithDisableSourceMaps, parser.IsModule)
			if prsderr != nil {
				if setchdprgm != nil {
					setchdprgm(nil, prsderr, nil)
				}
				perr = &parseerr{cde: cdes, err: prsderr}
				return
			}
			p, perr = ja.CompileAST(prsd, false)
			if perr != nil {
				if setchdprgm != nil {
					setchdprgm(nil, nil, perr)
				}
				perr = &parseerr{cde: cdes, err: prsderr}
				return
			}
			if setchdprgm != nil {
				setchdprgm(p, nil, nil)
			}
		}
	}

	return
}

func (vm *VM) Eval(a ...interface{}) (val interface{}, err error) {
	if vm != nil && vm.vm != nil {
		//var cdes = ""
		//var chdprgm *ja.Program = nil
		//var setchdprgm func(interface{}, error, error)
		var ai, ail = 0, len(a)

		var errfound func(...interface{}) error = nil
		for ai < ail {
			if errfoundd, errfounddok := a[ai].(func(...interface{}) error); errfounddok {
				if errfound == nil && errfoundd != nil {
					errfound = errfoundd
				}
				ail--
				a = append(a[:ai], a[ai+1:]...)
			} else {
				ai++
			}
		}

		if func() {
			p, perr := Compile(a...)
			if perr != nil {
				err = perr
			}
			gojaval, gojaerr := vm.vm.RunProgram(p)
			if gojaerr == nil {
				if gojaval != nil {
					val = gojaval.Export()
					return
				}
				return
			}

			err = gojaerr
		}(); err != nil {
			errfns := []func(...interface{}) error{}
			if vm.ErrPrint != nil {
				errfns = append(errfns, vm.ErrPrint)
			}
			if errfound != nil {
				errfns = append(errfns, errfound)
			}
			cdes := ""
			if prserr, _ := err.(*parseerr); prserr != nil {
				cdes = prserr.Code()
			}
			for _, ErrPrint := range errfns {
				func() {
					var linecnt = 1
					var errcdebuf = iorw.NewBuffer()
					errcdebuf.Print(fmt.Sprintf("%d: ", linecnt))
					defer errcdebuf.Close()
					var prvr = rune(0)
					for _, r := range cdes {
						if r == '\n' {
							linecnt++
							if prvr == '\r' {
								errcdebuf.WriteRune(prvr)
							}
							errcdebuf.WriteRune(r)
							errcdebuf.Print(fmt.Sprintf("%d: ", linecnt))
							prvr = 0
						} else {
							if r != '\r' {
								errcdebuf.WriteRune(r)
							}
						}
						prvr = r
					}
					ErrPrint("err:"+err.Error(), "\r\n", "err-code:"+errcdebuf.String())
				}()
			}
		}
	}
	return
}

func (vm *VM) Close() {
	if vm != nil {
		if vm.objmap != nil {
			if DisposeObject, objmap := vm.DisposeObject, vm.objmap; objmap != nil || DisposeObject != nil {
				if DisposeObject != nil {
					vm.DisposeObject = nil
					if objmap != nil {
						for objname, objval := range vm.objmap {
							vm.Remove(objname)
							DisposeObject(objname, objval)
						}
						vm.objmap = nil
					}
				}
				if objmap != nil {
					for objname := range vm.objmap {
						vm.Remove(objname)
					}
					vm.objmap = nil
				}
			}
			vm.objmap = nil
		}
		if vm.buffs != nil {
			for buf := range vm.buffs {
				buf.Close()
			}
			vm.buffs = nil
		}
		if gojavm := vm.vm; gojavm != nil {
			vm.vm = nil
		}
	}
}

var gobalMods *sync.Map

var adhocPrgm *ja.Program = nil

func LoadGlobalModule(modname string, a ...interface{}) {
	if _, ok := gobalMods.Load(modname); ok {

	} else {
		func() {
			var cdebuf = iorw.NewBuffer()
			defer cdebuf.Close()
			if prgmast, _ := ja.Parse(modname, cdebuf.String()); prgmast != nil {
				if prgm, _ := ja.CompileAST(prgmast, false); prgm != nil {
					gobalMods.Store(modname, prgm)
				}
			}
		}()
	}
}

func IncludeModule(vm *ja.Runtime, modname string) {
	if prgv, ok := gobalMods.Load(modname); ok {
		if prg, _ := prgv.(*ja.Program); prg != nil {
			vm.RunProgram(prg)
		}
	}
}

func init() {

	gobalMods = &sync.Map{}

	if adhocast, _ := ja.Parse(``, `_methods = (obj) => {
		let properties = new Set()
		let currentObj = obj
		Object.entries(currentObj).forEach((key)=>{
			key=(key=(key+"")).indexOf(",")>0?key.substring(0,key.indexOf(',')):key;
			if (typeof currentObj[key] === 'function') {
				var item=key;
				properties.add(item);
			}
		});
		if (properties.size===0) {
			do {
				Object.getOwnPropertyNames(currentObj).map(item => properties.add(item))
			} while ((currentObj = Object.getPrototypeOf(currentObj)))
		}
		return [...properties.keys()].filter(item => typeof obj[item] === 'function')
	}
	
	_fields = (obj) => {
		let properties = new Set()
		let currentObj = obj
		Object.entries(currentObj).forEach((key)=>{
			key=(key=(key+"")).indexOf(",")>0?key.substring(0,key.indexOf(',')):key;
			if (typeof currentObj[key] !== 'function') {
				var item=key;
				properties.add(item);
			}
		});
		if (properties.size===0) {
			do {
				Object.getOwnPropertyNames(currentObj).map(item => properties.add(item))
			} while ((currentObj = Object.getPrototypeOf(currentObj)))
		}
		return [...properties.keys()].filter(item => item!=='__proto__' && typeof obj[item] !== 'function')
	}`); adhocast != nil {
		adhocPrgm, _ = ja.CompileAST(adhocast, false)
	}

}
