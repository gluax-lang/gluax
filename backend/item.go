package codegen

import (
	"log"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (cg *Codegen) genItem(item ast.Item) {
	switch it := item.(type) {
	case *ast.Import, *ast.Struct:
		return
	case *ast.Let:
		cg.genLet(it)
	case *ast.Function:
		fun := it.Sem()
		name := cg.decorateFuncName(fun)
		cg.ln("%s = %s;", name, cg.genFunction(it.Sem()))
	}
	cg.ln("")
}

func (cg *Codegen) genImport(it *ast.Import) {
	if it.Path.Raw == "globals*" {
		return
	}
	imp := cg.Analysis.Scope.GetImport(it.As.Raw)
	if imp == nil {
		log.Println("import not found", it.Path.Raw)
		return
	}
	cg.ln("%s(\"%s\");", RUN_IMPORT, toHexEscapedLiteral(imp.Path))
}
