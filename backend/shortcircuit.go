package codegen

import (
	"github.com/gluax-lang/gluax/frontend/ast"
)

// most of this code was done with AI, this is because luajit does some hacky optimizations to do short-circuit
// evaluations, previously when I wrote it, it was unoptimized version and I didn't like how it would work
//
// this is done because gluax has statements as expressions unlike lua
// which causes issues when you do something like `if call() && if true {true} else {false} {}`
// if we don't desugar, it will generate:
// local temp; if true then temp = true else temp = false end; if call() and temp then ...
// which is wrong, that's why we need to manually implement short-circuit evaluation
// this can be improved to be 1:1 to what luajit produces, but this will be done when
// I or someone understands how luajit does it (luajit does everything in a single pass so it's complicated to understand)

type Instr any

type (
	Label struct {
		Name string
	}
	Goto struct {
		Label string
	}
	CondGoto struct {
		Expr       ast.Expr
		TrueLabel  string
		FalseLabel string
	}
	AssignLiteral struct {
		Dest  string
		Value bool
	}
)

type LogicalGroup struct {
	Op    ast.BinaryOp
	Exprs []any // ast.Expr or *LogicalGroup
}

func groupLogicalExpr(e ast.Expr) any {
	if e.Kind() == ast.ExprKindBinary &&
		(e.Binary().Op == ast.BinaryOpLogicalAnd || e.Binary().Op == ast.BinaryOpLogicalOr) {

		group := &LogicalGroup{Op: e.Binary().Op}
		appendExpr := func(expr any) {
			if sg, ok := expr.(*LogicalGroup); ok && sg.Op == group.Op {
				group.Exprs = append(group.Exprs, sg.Exprs...)
			} else {
				group.Exprs = append(group.Exprs, expr)
			}
		}
		appendExpr(groupLogicalExpr(e.Binary().Left))
		appendExpr(groupLogicalExpr(e.Binary().Right))
		return group
	}
	return e
}

func (cg *Codegen) buildIR(node any, dest, tLab, fLab string) []Instr {
	switch n := node.(type) {
	case *LogicalGroup:
		return cg.buildGroup(n, dest, tLab, fLab)
	case ast.Expr:
		// leaf: evaluate into dest, then branch
		return []Instr{
			CondGoto{Expr: n, TrueLabel: tLab, FalseLabel: fLab},
		}
	}
	return nil
}

func (cg *Codegen) buildGroup(n *LogicalGroup, dest, tLab, fLab string) []Instr {
	if len(n.Exprs) == 0 {
		// nothing: fall through to fLab
		return nil
	}
	if len(n.Exprs) == 1 {
		// single: just forward
		return cg.buildIR(n.Exprs[0], dest, tLab, fLab)
	}

	head := n.Exprs[0]
	rest := &LogicalGroup{Op: n.Op, Exprs: n.Exprs[1:]}
	mid := cg.temp() + "_mid"

	var ir []Instr
	if n.Op == ast.BinaryOpLogicalAnd {
		// a && b: if a true -> mid, else -> fLab
		ir = cg.buildIR(head, dest, mid, fLab)
		ir = append(ir, Label{mid})
		// then b && c... -> tLab/fLab
		ir = append(ir, cg.buildIR(rest, dest, tLab, fLab)...)
	} else {
		// OR: a || b: if a true -> tLab, else -> mid
		ir = cg.buildIR(head, dest, tLab, mid)
		ir = append(ir, Label{mid})
		// then b || c... -> tLab/fLab
		ir = append(ir, cg.buildIR(rest, dest, tLab, fLab)...)
	}
	return ir
}

// Emit all instructions, optimizing away unneeded labels/gotos
func (cg *Codegen) emitIR(ir []Instr, dest string) {
	labelRefs := countLabelRefs(ir)

	for i := range ir {
		inst := ir[i]

		switch v := inst.(type) {
		case Label:
			// Only emit labels that are referenced
			if labelRefs[v.Name] > 0 {
				cg.ln("::%s::", v.Name)
			}
		case Goto:
			// skip redundant jumps to immediately-following labels
			if i+1 < len(ir) && isLabel(ir[i+1], v.Label) {
				continue
			}
			cg.ln("goto %s", v.Label)
		case CondGoto:
			cg.emitCondGoto(v, dest, ir, i, labelRefs)
		}
	}
}

func (cg *Codegen) emitCondGoto(v CondGoto, dest string, ir []Instr, i int, labelRefs map[string]int) {
	cg.ln("%s = %s", dest, cg.genExprX(v.Expr))
	next := nextLabel(ir, i+1)

	trueFall := v.TrueLabel == next
	falseFall := v.FalseLabel == next

	switch {
	case trueFall && !falseFall:
		cg.ln("if not %s then goto %s end", dest, v.FalseLabel)
		labelRefs[v.TrueLabel]-- // we skipped emitting a jump to trueLabel

	case falseFall && !trueFall:
		cg.ln("if %s then goto %s end", dest, v.TrueLabel)
		labelRefs[v.FalseLabel]--

	case !falseFall && !trueFall:
		cg.ln("if %s then goto %s else goto %s end", dest, v.TrueLabel, v.FalseLabel)

	default:
		// Both true and false fallthrough: no condition needed
		labelRefs[v.TrueLabel]--
		labelRefs[v.FalseLabel]--
	}
}

// Helper function: count references to labels in the IR
func countLabelRefs(ir []Instr) map[string]int {
	refs := make(map[string]int)
	for _, inst := range ir {
		switch v := inst.(type) {
		case Goto:
			refs[v.Label]++
		case CondGoto:
			refs[v.TrueLabel]++
			refs[v.FalseLabel]++
		}
	}
	return refs
}

func isLabel(instr Instr, name string) bool {
	lbl, ok := instr.(Label)
	return ok && lbl.Name == name
}

func nextLabel(ir []Instr, idx int) string {
	for i := idx; i < len(ir); i++ {
		if lbl, ok := ir[i].(Label); ok {
			return lbl.Name
		}
		if _, isGoto := ir[i].(Goto); !isGoto {
			break
		}
	}
	return ""
}

func (cg *Codegen) genShortCircuitExpr(e ast.Expr) string {
	dest := cg.temp()
	cg.ln("local %s;", dest)
	cg.ln("do")
	cg.pushIndent()
	tLabel, fLabel := cg.temp()+"_true", cg.temp()+"_false"
	endL := cg.temp() + "_end"

	ir := cg.buildIR(groupLogicalExpr(e), dest, tLabel, fLabel)

	// Append the final “just go to end” blocks
	ir = append(ir,
		Label{fLabel}, Goto{endL},
		Label{tLabel}, Goto{endL},
		Label{endL},
	)

	// Now optimize trivial labels that do nothing but `goto x`
	ir = mergeTrivialLabels(ir)

	// Emit the cleaned-up IR
	cg.emitIR(ir, dest)
	cg.popIndent()
	cg.ln("end")

	return dest
}

// mergeTrivialLabels finds any label that immediately does `goto SomeOther`
// and replaces references to that label with SomeOther, removing
// the original label from the IR if it has no other code behind it.
func mergeTrivialLabels(ir []Instr) []Instr {
	// Map of labels that can be redirected to other labels
	labelRedirects := buildLabelRedirectMap(ir)

	// Resolve label chains (X->Y->Z becomes X->Z)
	resolve := func(l string) string {
		for {
			next, ok := labelRedirects[l]
			if !ok || next == l {
				return l
			}
			l = next
		}
	}

	out := make([]Instr, 0, len(ir))
	skipNext := false

	for i := range ir {
		if skipNext {
			skipNext = false
			continue
		}

		switch v := ir[i].(type) {
		case Label:
			// If label is `Label{X}` followed by `Goto{...}` and
			// we had X => SomeOther, we skip them (they’re trivial).
			_, isTrivial := labelRedirects[v.Name]
			if (isTrivial && i+1 < len(ir)) && isJustGoto(ir[i+1]) {
				// we skip both the label and the next goto
				skipNext = true
				continue
			}
			// otherwise keep the label, but also check if it’s been redirected
			// (if some label points to this label).
			newName := resolve(v.Name)
			// If it resolves to itself, keep it
			if newName == v.Name {
				out = append(out, v)
			} else {
				// it resolves away to something else, so skip
			}

		case Goto:
			// rewrite references
			v.Label = resolve(v.Label)
			out = append(out, v)

		case CondGoto:
			v.TrueLabel = resolve(v.TrueLabel)
			v.FalseLabel = resolve(v.FalseLabel)
			out = append(out, v)

		default:
			out = append(out, v)
		}
	}

	return out
}

func isJustGoto(i Instr) bool {
	_, ok := i.(Goto)
	return ok
}

func buildLabelRedirectMap(ir []Instr) map[string]string {
	redirects := make(map[string]string)
	for i := range len(ir) - 1 {
		if lbl, ok := ir[i].(Label); ok {
			if g, ok := ir[i+1].(Goto); ok {
				// We have ::X:: then goto Y
				// Mark that X is effectively an alias for Y
				redirects[lbl.Name] = g.Label
			}
		}
	}
	return redirects
}
