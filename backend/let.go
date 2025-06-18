package codegen

import (
	"fmt"
	"strings"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (cg *Codegen) decorateLetName(l *ast.Let, n int) string {
	if l.IsGlobal() {
		return l.GlobalName(n)
	}
	name := l.Names[n]
	raw := name.Raw
	if l.IsItem {
		id := fmt.Sprintf("_%d", name.Span().ID)
		return cg.getPublic(LOCAL_PREFIX + raw + id)
	}
	return raw
}

func (cg *Codegen) genLetLHS(l *ast.Let) []string {
	lhs := make([]string, len(l.Names))
	for i := range l.Names {
		lhs[i] = cg.decorateLetName(l, i)
		if l.IsItem {
			lhs[i] += fmt.Sprintf(" --[[let %s]]", l.Names[i].Raw)
		}
		//  else if l.IsItem {
		// 	cg.currentTempScope().all = append(cg.currentTempScope().all, lhs[i])
		// }
	}
	return lhs
}

func (cg *Codegen) genLet(l *ast.Let) {
	if l.IsGlobal() {
		return
	}
	rhs := cg.genExprsLeftToRight(l.Values)
	lhs := cg.genLetLHS(l)
	if l.IsItem {
		cg.ln("%s = %s;", strings.Join(lhs, ", "), rhs)
	} else {
		cg.ln("local %s = %s;", strings.Join(lhs, ", "), rhs)
	}
}
