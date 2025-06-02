package codegen

import (
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
		if !it.Public {
			cg.currentTempScope().all = append(cg.currentTempScope().all, name)
		}
		cg.ln("%s = %s;", name, cg.genFunction(it.Sem()))
	}
	cg.ln("")
}

func (cg *Codegen) genImport(it *ast.Import) {
	cg.ln("%s(%s);", RUN_IMPORT, pathToLuaString(it.SafePath))
}
