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
		name := it.Name.Raw
		if it.Public {
			fun := it.Sem()
			name = cg.decorateFuncName(fun)
		}
		cg.ln("%s = %s;", name, cg.genFunction(it.Sem()))
	}
	cg.ln("")
}

func (cg *Codegen) genImport(it *ast.Import) {
	imp := cg.Analysis.Scope.GetImport(it.Path.Raw)
	if imp == nil {
		if it.Path.Raw != "globals*" {
			log.Println("import not found", it.Path.Raw)
		}
		return
	}
	cg.ln("%s(\"%s\");", RUN_IMPORT, toHexEscapedLiteral(imp.Path))
}
