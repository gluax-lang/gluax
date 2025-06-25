package codegen

import (
	"fmt"
	"strings"

	"github.com/gluax-lang/gluax/frontend"
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (cg *Codegen) decorateTraitName_internal(tr *ast.Trait, class *ast.SemClass) string {
	raw := tr.Name.Raw
	var sb strings.Builder
	sb.WriteString(frontend.TRAIT_PREFIX)
	sb.WriteString(raw)
	sb.WriteString(fmt.Sprintf("_%d", tr.Span().ID))
	if class != nil {
		sb.WriteString(cg.decorateClassName_internal(class))
	}
	return sb.String()
}

func (cg *Codegen) decorateTraitName(tr *ast.Trait, class *ast.SemClass) string {
	raw := tr.Name.Raw
	var sb strings.Builder
	sb.WriteString(frontend.TRAIT_PREFIX)
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
			cg.ln("%s = %s,", hMethod.Def.Name.Raw, cg.genFunction(hMethod))
		}

		cg.popIndent()
		cg.ln("};")
	}
}
