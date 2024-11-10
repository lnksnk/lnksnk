package es

import (
	"github.com/lnksnk/lnksnk/es/ast"
	"github.com/lnksnk/lnksnk/es/unistring"
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
		for _, nmdimprt := range _imprtdcl.importclause.namedimports {
			nmdImprts = append(nmdImprts, []string{string(nmdimprt.identifier), string(nmdimprt.alias)})
		}
		importModule(string(_imprtdcl.fromclause), nmdImprts)
		vm.pc++
	}
}

func importDeclFromAst(expr *ast.ImportDeclaration) (imprtdcl _importDecl) {
	if expr == nil || expr.FromClause == nil {
		return
	}
	var namedimports []_namedImport
	for _, nmdimprt := range expr.ImportClause.NamedImports.ImportsList {
		namedimports = append(namedimports, _namedImport{identifier: nmdimprt.IdentifierName, alias: nmdimprt.Alias})
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
