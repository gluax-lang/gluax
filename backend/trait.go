package codegen

import (
	"fmt"
	"strings"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (cg *Codegen) decorateTraitName(tr *ast.Trait, class *ast.SemClass) string {
	raw := tr.Name.Raw
	var sb strings.Builder
	sb.WriteString(TRAIT_PREFIX)
	sb.WriteString(raw)
	sb.WriteString(fmt.Sprintf("_%d", tr.Span().ID))
	if class != nil {
		sb.WriteString(cg.decorateClassName_internal(class))
	}
	baseName := sb.String()
	var comment string
	if class != nil {
		comment = fmt.Sprintf("impl %s for %s", raw, class.Def.Name.Raw)
	} else {
		comment = fmt.Sprintf("trait %s", raw)
	}
	return cg.getPublic(baseName) + fmt.Sprintf(" --[[%s]]", comment)
}

func (cg *Codegen) genTrait(tr *ast.Trait) {
	/*
		local trait = {
			func_name = function(v, ...)
				local obj = v[1]
				local vtable = v[2]
				return vtable.func_name(obj, ...)
			end
		}
	*/
	dTName := cg.decorateTraitName(tr, nil)
	cg.ln("%s = {", dTName)
	cg.pushIndent()
	for _, m := range tr.Methods {
		mName := m.Name.Raw
		params := cg.genFunctionParams(m)
		self := params[0]
		cg.ln("%s = function(%s)", mName, strings.Join(params, ", "))
		cg.pushIndent()
		params[0] = self + "[1]" // self is the first parameter, which is the value
		cg.ln("return %s[2].%s(%s)", self, mName, strings.Join(params, ", "))
		cg.popIndent()
		cg.ln("end,")
	}
	cg.popIndent()
	cg.ln("};")
}

func (cg *Codegen) genTraitImpl(tr *ast.SemTrait) {
	classesAndMethods := cg.Analysis.GetClassesImplementingTrait(tr)

	for class, methods := range classesAndMethods {
		if !class.IsFullyConcrete() {
			continue
		}

		dTName := cg.decorateTraitName(tr.Def, class)

		cg.ln("%s = {", dTName)
		cg.pushIndent()

		for _, m := range methods {
			hMethod := cg.Analysis.HandleClassMethod(class, m, true)
			cg.ln("%s = %s,", hMethod.Def.Name.Raw, cg.genFunction(&hMethod))
		}

		cg.popIndent()
		cg.ln("};")
	}
}
