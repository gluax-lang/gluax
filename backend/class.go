package codegen

import (
	"fmt"
	"strings"

	"github.com/gluax-lang/gluax/frontend"
	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/sema"
)

func (cg *Codegen) decorateClassName_internal(cls *ast.SemClass) string {
	if !cg.markUsed(cls) {
		cg.generateClass(cls)
	}
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

func (cg *Codegen) classFuncUsedName(clss *ast.SemClass, methodName string) string {
	return fmt.Sprintf("%s.%s", cg.decorateClassName_internal(clss), methodName)
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
			if method.Def.Body != nil {
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
	cg.genClassFuncs(st, methods)
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

func (cg *Codegen) genClassFuncs(clss *ast.SemClass, funcs map[string]*sema.SemFunction) {
	for name, method := range funcs {
		if method.Def.Body == nil {
			continue
		}
		if !cg.isMarkedUsed(cg.classFuncUsedName(clss, name)) {
			continue
		}
		// we need to handle it with body, to make sure body calls are generated correctly
		hMethod := cg.Analysis.HandleClassMethod(clss, method, true)
		if rename := method.Attributes().GetString("rename_to"); rename != nil {
			name = *rename
		}
		cg.ln("%s = %s,", name, cg.genFunction(hMethod))
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
