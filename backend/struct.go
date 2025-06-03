package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (cg *Codegen) decorateStName_internal(st *ast.SemStruct) string {
	var sb strings.Builder
	var emit func(s *ast.SemStruct)
	emit = func(s *ast.SemStruct) {
		if sb.Len() == 0 {
			sb.WriteString(STRUCT_PREFIX)
		}
		sb.WriteString(s.Def.Name.Raw)
		// we don't need an id for builtin types, because they are always unique everywhere
		if s.Def.Public && !ast.IsBuiltinType(s.Def.Name.Raw) {
			id := fmt.Sprintf("_%d", s.Def.Span().ID)
			sb.WriteString(id)
		}
		for _, g := range s.Generics.Params {
			sb.WriteByte('_')
			switch {
			case g.IsStruct():
				emit(g.Struct())
			case g.IsGeneric():
				panic("THIS SHOULD NOT HAPPEN WITH NEW VERSION OF GLUAX")
				sb.WriteString(g.Generic().Ident.Raw)
			case g.IsFunction():
				panic("TODO: handle function generics in struct names")
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

func (cg *Codegen) decorateStName(st *ast.SemStruct) string {
	{
		raw := st.Def.Name.Raw
		if st.Def.Public && st.Def.IsGlobalDef {
			if rename_to := st.Def.Attributes.GetString("rename_to"); rename_to != nil {
				return *rename_to
			}
			return raw
		}
	}
	baseName := cg.decorateStName_internal(st)
	if st.Def.Public {
		return cg.getPublic(baseName) + fmt.Sprintf(" --[[struct: %s]]", st.String())
	}
	return baseName + fmt.Sprintf(" --[[struct: %s]]", st.String())
}

func structHeaders(cg *Codegen) {

}

func (cg *Codegen) generateStruct(st *ast.SemStruct) {
	for _, g := range st.Generics.Params {
		if g.IsGeneric() {
			return // we don't generate structs with generics, because they are not concrete types
		}
	}
	name := cg.decorateStName(st)
	{
		if _, ok := cg.generatedStructs[name]; ok {
			return
		}
		cg.generatedStructs[name] = struct{}{}
	}
	if !st.Def.Public {
		cg.currentTempScope().all = append(cg.currentTempScope().all, name)
	}
	cg.ln("%s = {", name)
	cg.pushIndent()
	stMethods := cg.Analysis.State.StructsMethods[st.Def]
	if stMethods != nil {
		for e := range stMethods.Methods {
			method, exists := stMethods.GetStructMethod(e, st.Generics.Params)
			if exists {
				// we need to handle it with body, to make sure body calls are generated correctly
				method = cg.Analysis.HandleStructMethod(st, method, true)
				cg.ln("%s = %s,", e, cg.genFunction(&method))
			}
		}
	}
	cg.popIndent()
	cg.ln("}\n")
}

func (cg *Codegen) genStructInit(si *ast.ExprStructInit, st *ast.SemStruct) string {
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

	tempNames, _ := cg.genExprsToLocals(exprs, false)

	for i := range fieldEvals {
		fieldEvals[i].Temp = tempNames[i]
	}

	toSetTo := cg.decorateStName(st)

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
	st := toIndexTy.Struct()
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
