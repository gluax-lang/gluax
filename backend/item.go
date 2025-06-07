package codegen

import (
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (cg *Codegen) genImport(it *ast.Import) {
	cg.ln("%s(%s);", RUN_IMPORT, pathToLuaString(it.SafePath))
}
