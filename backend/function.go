package codegen

import (
	"fmt"
	"strings"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (cg *Codegen) decorateFuncName(f *ast.SemFunction) string {
	raw := f.Def.Name.Raw
	if f.Def.Public && f.Def.IsGlobalDef {
		attrs := f.Def.Attributes
		for _, attr := range attrs {
			if attr.Key.Raw == "rename_to" {
				if attr.IsInputString() {
					return attr.String.Raw
				}
			}
		}
		return raw
	}
	var sb strings.Builder
	sb.WriteString(FUNC_PREFIX)
	sb.WriteString(f.Def.Name.Raw)
	if f.Def.Public {
		id := fmt.Sprintf("_%d", f.Def.Span().ID)
		sb.WriteString(id)
	}
	baseName := sb.String()
	if f.Def.Public {
		return cg.getPublic(baseName) + fmt.Sprintf(" --[[%s]]", f.String())
	}
	return baseName + fmt.Sprintf(" --[[%s]]", f.String())
}

func (cg *Codegen) genFunction(f *ast.SemFunction) string {
	def := f.Def
	oldBuf := cg.newBuf()
	cg.writeString("function(")
	for i, p := range def.Params {
		if i > 0 {
			cg.writeString(", ")
		}
		cg.writeString(p.String())
	}
	cg.writeByte(')')
	cg.writeByte('\n')
	cg.pushIndent()

	cg.pushTempScope()

	// make another buffer for the body, so we can use it for the return value
	bodyBuf := cg.newBuf()
	if f.HasVarargReturn() {
		cg.genBlockX(def.Body, BlockNone)
	} else {
		value := cg.genBlockX(def.Body, BlockNone)
		if f.Def.Errorable {
			cg.ln("return nil, %s;", value)
		} else {
			cg.ln("return %s;", value)
		}
	}
	bodySnippet := cg.restoreBuf(bodyBuf)
	cg.emitTempLocals()
	cg.writeString(bodySnippet)
	cg.popIndent()
	cg.writeIndent()
	cg.writeString("end")
	snippet := cg.restoreBuf(oldBuf)
	return snippet
}
