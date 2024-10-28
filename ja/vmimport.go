package ja

import "github.com/lnksnk/lnksnk/ja/unistring"

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
		for _, nmdimprt := range _imprtdcl.importclause.namedimports {
			nmdImprts = append(nmdImprts, []string{string(nmdimprt.identifier), string(nmdimprt.alias)})
		}
		importModule(string(_imprtdcl.fromclause), nmdImprts)
		vm.pc++
	}
}
