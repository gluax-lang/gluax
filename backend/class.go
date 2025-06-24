package codegen

import (
	"fmt"
	"strings"

	"github.com/gluax-lang/gluax/frontend"
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (cg *Codegen) decorateClassName_internal(cls *ast.SemClass) string {
	var sb strings.Builder
	sb.WriteString(frontend.CLASS_PREFIX)
	sb.WriteString(cls.Def.Name.Raw)
	sb.WriteString(fmt.Sprintf("_%d", cls.Def.Span().ID))
	sb.WriteString(fmt.Sprintf("_%p", cls))
	return sb.String()
}

func (cg *Codegen) decorateClassName(st *ast.SemClass) string {
	if st.IsGlobal() {
		// return st.GlobalName()
	}
	baseName := cg.decorateClassName_internal(st)
	return cg.getPublic(baseName) + fmt.Sprintf(" --[[class %s]]", st.String())
}

func classHeaders(cg *Codegen) {

}

func (cg *Codegen) generateClass(st *ast.SemClass) {
	if !st.IsFullyConcrete() {
		return // we don't generate classes with generics, because they will never be used
	}
	if st.IsNilable() || st.IsAnyFunc() {
		// don't generate phantom types
		return
	}
	methods := cg.Analysis.FindAllClassMethods(st)
	if st.IsGlobal() {
		// global classes are just phantom, they exist in lua world!
		// NEW: UNLESS WE ADDED FUNCTIONS TO THEM HAHA
		canGenerate := false
		for _, method := range methods {
			if st.CanGenerateMethod(&method.Def) && !method.IsGlobal() {
				canGenerate = true
				break
			}
		}
		if !canGenerate {
			// no local methods, so we don't generate anything
			return
		}
	}
	name := cg.decorateClassName(st)
	{
		if _, ok := cg.generatedClasses[name]; ok {
			return
		}
		cg.generatedClasses[name] = struct{}{}
	}
	cg.ln("%s = {", name)
	cg.pushIndent()
	cg.ln("%s = true,", frontend.CLASS_MARKER_PREFIX)
	for name, method := range methods {
		if !st.CanGenerateMethod(&method.Def) || method.IsGlobal() {
			continue
		}
		// we need to handle it with body, to make sure body calls are generated correctly
		hMethod := cg.Analysis.HandleClassMethod(st, method, true)
		cg.ln("%s = %s,", name, cg.genFunction(&hMethod))
	}
	cg.popIndent()
	cg.ln("};")
	if !st.Attributes().Has("no__index", "no_metatable") && !st.IsGlobal() {
		cg.ln("%s.__index = %s;", name, name)
		if st.Super != nil {
			superName := cg.decorateClassName(st.Super)
			cg.ln("setmetatable(%s, %s);", name, superName)
		}
	}
}

func getClassFieldIndex(clss *ast.SemClass, fieldName string) string {
	if clss.Attributes().Has("named_fields") {
		return fmt.Sprintf("[%q]", fieldName)
	}
	return fmt.Sprintf("[%d]--[[%s]]", clss.GetFieldIndex(fieldName), fieldName)
}

func (cg *Codegen) genClassInit(si *ast.ExprClassInit, st *ast.SemClass) string {
	exprs := make([]ast.Expr, len(si.Fields))
	for i, f := range si.Fields {
		exprs[i] = f.Value
	}

	tempNames := cg.genExprsToStrings(exprs)
	toSetTo := cg.decorateClassName(st)

	var sb strings.Builder
	sb.WriteString("setmetatable({")

	for i, f := range si.Fields {
		if i > 0 {
			sb.WriteString(", ")
		}
		fieldIndex := getClassFieldIndex(st, f.Name.Raw)
		sb.WriteString(fmt.Sprintf("%s=%s", fieldIndex, tempNames[i]))
	}

	sb.WriteString(fmt.Sprintf("}, %s)", toSetTo))
	return sb.String()
}

func (cg *Codegen) genDotAccess(expr *ast.DotAccess, toIndex string, toIndexTy ast.SemType) string {
	st := toIndexTy.Class()
	// Use numeric index for field access
	return fmt.Sprintf("%s%s", toIndex, getClassFieldIndex(st, expr.Name.Raw))
}
