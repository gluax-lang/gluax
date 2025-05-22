package codegen

import (
	"fmt"
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
				sb.WriteString(g.Generic().Ident.Raw)
			case g.IsFunction():
				f := g.Function()
				sb.WriteString(cg.decorateFuncName(&f))
			default:
				panic("not yet implemented")
			}
		}
	}
	emit(st)
	return sb.String()
}

func (cg *Codegen) decorateStName(st *ast.SemStruct) string {
	baseName := cg.decorateStName_internal(st)
	if st.Def.Public {
		return cg.getPublic(baseName) + fmt.Sprintf(" --[[struct: %s]]", st.String())
	}
	return baseName + fmt.Sprintf(" --[[struct: %s]]", st.String())
}

func structSetObjFieldCode(st *ast.SemStruct, obj, key, value string) string {
	return fmt.Sprintf(`%s[%s][%s] = %s`, STRUCT_OBJ_FIELDS, obj, key, value)
}

func structGetObjFieldCode(st *ast.SemStruct, obj, key string) string {
	return fmt.Sprintf(`%s[%s].%s`, STRUCT_OBJ_FIELDS, obj, key)
}

func structIsObjInstanceCode(obj, structName string) string {
	return fmt.Sprintf(`%s[%s] == %s`, STRUCT_OBJ_INSTANCES, obj, structName)
}

func structHeaders(cg *Codegen) {
	cg.ln("local %s = setmetatable({}, { __mode = \"k\" });", STRUCT_OBJ_INSTANCES)
	cg.ln("local %s = setmetatable({}, { __mode = \"k\" });", STRUCT_OBJ_FIELDS)
	cg.ln("local %s = function(struct, fields)", STRUCT_NEW)
	cg.pushIndent()
	cg.ln("local obj = {}")
	cg.ln("%s[obj] = struct", STRUCT_OBJ_INSTANCES)
	cg.ln("%s[obj] = fields", STRUCT_OBJ_FIELDS)
	cg.ln("return obj")
	cg.popIndent()
	cg.ln("end;")
}

func (cg *Codegen) generateStruct(st *ast.SemStruct) {
	name := cg.decorateStName(st)
	if st.Def.Public {
		if _, ok := cg.generatedStructs[name]; ok {
			return
		}
		cg.generatedStructs[name] = struct{}{}
	}
	cg.ln("%s = {", name)
	cg.pushIndent()
	for _, m := range st.Methods {
		cg.ln("%s = %s,", m.Def.Name.Raw, cg.genFunction(&m))
	}
	cg.popIndent()
	cg.ln("}\n")
}

func (cg *Codegen) genStructInit(si *ast.ExprStructInit, st *ast.SemStruct) string {
	var sb strings.Builder

	sb.WriteString("(")

	rhs := make([]string, len(si.Fields))
	for i, f := range si.Fields {
		rhs[i] = cg.genExpr(f.Value)
	}

	toCall := cg.decorateStName(st)

	sb.WriteString(fmt.Sprintf("%s(%s, {", STRUCT_NEW, toCall))

	for i, f := range si.Fields {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(f.Name.Raw)
		sb.WriteString(" = ")
		sb.WriteString(rhs[i])
	}

	sb.WriteString("}))")

	return sb.String()
}

func (cg *Codegen) genPathCall(call *ast.ExprPathCall) string {
	if !call.IsStructMethod() {
		funcTy := call.ImportedFunc()
		fun := funcTy.Function()
		name := cg.decorateFuncName(&fun)
		callCode := cg.genCall(&call.Call, name, funcTy)
		return callCode
	}
	st := call.Struct()
	name := cg.decorateStName(st)
	method, _ := st.GetMethod(call.MethodName.Raw)
	methodTy := ast.NewSemType(method, call.Span())
	callCode := cg.genCall(&call.Call, name+"."+call.MethodName.Raw, methodTy)
	return callCode
}

func (cg *Codegen) genDotAccess(expr *ast.DotAccess, toIndex string, toIndexTy ast.SemType) string {
	return structGetObjFieldCode(toIndexTy.Struct(), toIndex, expr.Name.Raw)
}
