package es

import (
	"fmt"

	"github.com/lnksnk/lnksnk/es/ast"
	"github.com/lnksnk/lnksnk/es/parser"
	"github.com/lnksnk/lnksnk/es/unistring"
	"github.com/lnksnk/lnksnk/fs"
	"github.com/lnksnk/lnksnk/ioext"
)

type _importClause struct {
	defaultbind   unistring.String
	namespacebind unistring.String
	namedimports  []_namedImport
}

type _namedImport struct {
	identifier unistring.String
	alias      unistring.String
}

type _importDecl struct {
	specifier    unistring.String
	importclause _importClause
	fromclause   unistring.String
}

func (_imprtdcl _importDecl) exec(vm *vm) {
	if importModule := vm.r.importModule; importModule != nil {
		var nmdImprts [][]string
		if _imprtdcl.importclause.defaultbind != "" {
			nmdImprts = append(nmdImprts, []string{"default", _imprtdcl.importclause.defaultbind.String()})
		}
		for _, nmdimprt := range _imprtdcl.importclause.namedimports {
			nmdImprts = append(nmdImprts, []string{string(nmdimprt.identifier), string(nmdimprt.alias)})
		}
		importModule(string(_imprtdcl.fromclause), nmdImprts)
	}
	vm.pc++
}

func importDeclFromAst(expr *ast.ImportDeclaration) (imprtdcl _importDecl) {
	if expr == nil || expr.FromClause == nil {
		return
	}
	var namedimports []_namedImport
	if nmimprts := expr.ImportClause.NamedImports; nmimprts != nil {
		for _, nmdimprt := range nmimprts.ImportsList {
			namedimports = append(namedimports, _namedImport{identifier: nmdimprt.IdentifierName, alias: nmdimprt.Alias})
		}
	}
	var importclause = _importClause{}
	if expr.ImportClause.ImportedDefaultBinding != nil {
		importclause.defaultbind = expr.ImportClause.ImportedDefaultBinding.Name
	}

	if expr.ImportClause.NameSpaceImport != nil {
		importclause.namespacebind = expr.ImportClause.NameSpaceImport.ImportedBinding
	}
	importclause.namedimports = namedimports
	imprtdcl = _importDecl{fromclause: expr.FromClause.ModuleSpecifier, specifier: expr.ModuleSpecifier, importclause: importclause}
	return
}

func (rt *Runtime) EvaluateModRec(modrec interface{}) (exports *Object) {
	if m, mk := modrec.(ModuleRecord); mk {
		evalprms := m.Evaluate(rt)
		if evalprms.State() == PromiseStateFulfilled {
			exports = rt.NamespaceObjectFor(m)
		}
	}
	return
}

func CompileProgram(fsys fs.MultiFileSystem, resolveMod func(refscriptormod interface{}, modspecifier string) (rlsvdmodrec interface{}, rslvderr error), cde ...interface{}) (prgm interface{}, err error) {
	var prsd *ast.Program
	prsd, err = parser.ParseFile(nil, "", func() string {
		if len(cde) == 0 {
			return ""
		}
		if len(cde) == 1 {
			if s, sk := cde[0].(string); sk {
				return s
			}
			if int32r, int32k := cde[0].([]int32); int32k {
				return string(int32r)
			}
			if bf, bfk := cde[0].(*ioext.Buffer); bfk {
				if bf.Empty() {
					return ""
				}
				return bf.String()
			}
		}
		cdebf := ioext.NewBuffer(cde...)
		defer cdebf.Close()
		if cdebf.Empty() {
			return ""
		}
		return cdebf.String()
	}(), 0, parser.WithDisableSourceMaps, parser.IsModule)
	if err == nil {
		func() {
			defer func() {
				if x := recover(); x != nil {
					err, _ = x.(error)
				}
			}()
			if len(prsd.ExportEntries) == 0 {
				prgm, err = CompileAST(prsd, false)
				return
			}
			prgm, err = ModuleFromProgramAndLink(prsd, func(refscriptormod interface{}, modspecifier string) (mdrc interface{}, rslvderr error) {
				if mdrc, rslvderr = resolveMod(refscriptormod, modspecifier); rslvderr != nil {
					return
				}
				if mdrc, _ = mdrc.(ModuleRecord); mdrc != nil {
					return mdrc, nil
				}
				return nil, fmt.Errorf("unable to load specifier %s", modspecifier)
			})
		}()
	}
	return
}

func ParseAndLinkModule(specifier, src string, resolveModule func(refscriptormod interface{}, modspecifier string) (rlsvdmodrec interface{}, rslvderr error)) (modrec ModuleRecord, err error) {
	p, perr := ParseModule(specifier, src, func(referencingScriptOrModule interface{}, modspecifier string) (ModuleRecord, error) {
		if resolveModule != nil {
			rslvdmodrec, rslvderr := resolveModule(referencingScriptOrModule, modspecifier)
			if rslvderr != nil {
				return nil, rslvderr
			}
			if modrec, _ = rslvdmodrec.(ModuleRecord); modrec != nil {
				err = modrec.Link()
				return modrec, err
			}
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
	return
}

func RequireModuleExports(mod interface{}, rt *Runtime) (exports *Object) {
	m, _ := mod.(ModuleRecord)
	if m == nil {
		return
	}
	evalprms := m.Evaluate(rt)
	if evalprms.State() == PromiseStateFulfilled {
		exports = rt.NamespaceObjectFor(m)
	}
	return
}

func LinkModule(mod interface{}) (err error) {
	m, _ := mod.(ModuleRecord)
	if m == nil {
		return
	}
	return m.Link()
}

func ImportModule(mod interface{}, rt *Runtime, namedimports ...[][]string) bool {
	m, _ := mod.(ModuleRecord)
	if m == nil {
		return false
	}
	evalprms := m.Evaluate(rt)
	if evalprms.State() == PromiseStateFulfilled {
		nmspce := rt.NamespaceObjectFor(m)
		for _, nmdimprt := range namedimports {
			for _, imprtthis := range nmdimprt {
				if imprtthisl := len(imprtthis); imprtthisl > 0 {
					idntys := imprtthis[0]
					if idntys != "" {
						if imprtthisl > 1 {
							if aliass := imprtthis[1]; aliass != "" {
								rt.Set(aliass, nmspce.Get(idntys))
								continue
							}
						}
						rt.Set(idntys, nmspce.Get(idntys))
					}
				}
			}
		}
		return true
	}
	return false
}

func ModuleFromProgramAndLink(prgrm interface{}, resolveModule func(refscriptormod interface{}, modspecifier string) (rlsvdmodrec interface{}, rslvderr error)) (modrec ModuleRecord, err error) {
	if astpgrm, astpgrmk := prgrm.(*ast.Program); astpgrmk {
		modrec, err = ModuleFromAST(astpgrm, func(referencingScriptOrModule interface{}, modspecifier string) (ModuleRecord, error) {
			if resolveModule != nil {
				rslvdmodrec, rslvderr := resolveModule(referencingScriptOrModule, modspecifier)
				if rslvderr != nil {
					return nil, rslvderr
				}
				if modrec, _ = rslvdmodrec.(ModuleRecord); modrec != nil {
					err = modrec.Link()
					return modrec, err
				}
			}
			return nil, fmt.Errorf("unable to load specifier %s", modspecifier)
		})
		if err == nil && modrec != nil {
			err = modrec.Link()
		}
		return
	}

	return
}
