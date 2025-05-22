package codegen

import (
	"fmt"
	"strings"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (cg *Codegen) decorateLetName(l *ast.Let, n int) string {
	name := l.Names[n]
	raw := name.Raw
	if l.Public && l.IsGlobalDef {
		attrs := l.Attributes
		for _, attr := range attrs {
			if attr.Key.Raw == "rename_to" {
				if attr.IsInputString() {
					return attr.String.Raw
				}
			}
		}
		return raw
	}
	if l.Public {
		id := fmt.Sprintf("_%d", name.Span().ID)
		return cg.getPublic(LOCAL_PREFIX + raw + id)
	}
	return raw
}

func (cg *Codegen) genLet(l *ast.Let) {
	rhs := make([]string, len(l.Values))
	for i, v := range l.Values {
		rhs[i] = cg.genExpr(v)
	}
	lhs := make([]string, len(l.Names))
	for i := range l.Names {
		lhs[i] = cg.decorateLetName(l, i)
		if l.Public {
			lhs[i] += fmt.Sprintf(" --[[%s]]", l.Names[i].Raw)
		}
	}
	if l.Public {
		cg.ln("%s = %s;", strings.Join(lhs, ", "), strings.Join(rhs, ", "))
	} else {
		cg.ln("local %s = %s;", strings.Join(lhs, ", "), strings.Join(rhs, ", "))
	}
}
