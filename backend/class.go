package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (cg *Codegen) decorateClassName_internal(st *ast.SemClass) string {
	var sb strings.Builder
	var emit func(s *ast.SemClass)
	emit = func(s *ast.SemClass) {
		if sb.Len() == 0 {
			sb.WriteString(CLASS_PREFIX)
		}
		sb.WriteString(s.Def.Name.Raw)
		id := fmt.Sprintf("_%d", s.Def.Span().ID)
		sb.WriteString(id)
		for _, g := range s.Generics.Params {
			sb.WriteByte('_')
			switch {
			case g.IsClass():
				emit(g.Class())
			case g.IsGeneric():
				panic("THIS SHOULD NOT HAPPEN WITH NEW VERSION OF GLUAX")
				sb.WriteString(g.Generic().Ident.Raw)
			case g.IsFunction():
				panic("TODO: handle function generics in class names")
				// f := g.Function()
				// sb.WriteString(cg.decorateFuncName(&f))
			case g.IsUnreachable():
				sb.WriteString(UNREACHABLE_PREFIX)
			default:
				panic("not yet implemented")
			}
		}
	}
	emit(st)
	return sb.String()
}

func (cg *Codegen) decorateClassName(st *ast.SemClass) string {
	{
		raw := st.Def.Name.Raw
		if st.Def.Public && st.Def.IsGlobalDef {
			if rename_to := st.Def.Attributes.GetString("rename_to"); rename_to != nil {
				return *rename_to
			}
			return raw
		}
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
	name := cg.decorateClassName(st)
	{
		if _, ok := cg.generatedClasses[name]; ok {
			return
		}
		cg.generatedClasses[name] = struct{}{}
	}
	cg.ln("%s = {", name)
	cg.pushIndent()
	cg.ln("%s = true,", CLASS_MARKER_PREFIX)
	for name, method := range cg.Analysis.FindAllClassMethods(st) {
		// we need to handle it with body, to make sure body calls are generated correctly
		hMethod := cg.Analysis.HandleClassMethod(st, *method, true)
		cg.ln("%s = %s,", name, cg.genFunction(&hMethod))
	}
	cg.popIndent()
	cg.ln("};")
	if !st.Def.Attributes.Has("no__index") {
		cg.ln("%s.__index = %s;", name, name)
		if st.Super != nil {
			superName := cg.decorateClassName(st.Super)
			cg.ln("setmetatable(%s, %s);", name, superName)
		}
	}
}

func (cg *Codegen) genClassInit(si *ast.ExprClassInit, st *ast.SemClass) string {
	var sb strings.Builder

	type fieldEval struct {
		Name string
		Id   int
		Temp string
	}

	fieldEvals := make([]fieldEval, len(si.Fields))
	exprs := make([]ast.Expr, len(si.Fields))

	for i, f := range si.Fields {
		// Find the field definition to get its Id
		fieldId := 0
		for _, def := range st.Def.Fields {
			if def.Name.Raw == f.Name.Raw {
				fieldId = def.Id
				break
			}
		}
		fieldEvals[i] = fieldEval{
			Name: f.Name.Raw,
			Id:   fieldId,
		}
		exprs[i] = f.Value
	}

	tempNames := cg.genExprsToTempVars(exprs)

	for i := range fieldEvals {
		fieldEvals[i].Temp = tempNames[i]
	}

	toSetTo := cg.decorateClassName(st)

	// Sort by field Id for table initialization
	sorted := make([]fieldEval, len(fieldEvals))
	copy(sorted, fieldEvals)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Id < sorted[j].Id
	})

	sb.WriteString("setmetatable({")
	for i, fe := range sorted {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fe.Temp)
		sb.WriteString(fmt.Sprintf("--[[%s]]", fe.Name))
	}
	sb.WriteString(fmt.Sprintf("}, %s)", toSetTo))

	return sb.String()
}

func (cg *Codegen) genDotAccess(expr *ast.DotAccess, toIndex string, toIndexTy ast.SemType) string {
	st := toIndexTy.Class()
	fieldId := 0
	for _, def := range st.Def.Fields {
		if def.Name.Raw == expr.Name.Raw {
			fieldId = def.Id
			break
		}
	}
	// Use numeric index for field access
	return fmt.Sprintf("%s[%d--[[%s]]]", toIndex, fieldId, expr.Name.Raw)
}
