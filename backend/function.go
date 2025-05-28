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

func (cg *Codegen) getCallArgs(call *ast.Call, toCall string) string {
	_, args := cg.genExprsToLocals(call.Args, true)
	if call.Method != nil {
		if args == "" {
			return toCall
		} else {
			return toCall + ", " + args
		}
	} else {
		return args
	}
}

func (cg *Codegen) genInlineCall(call *ast.Call, fun ast.SemFunction, toCall string) string {
	// Inline the function body
	cg.ln("do -- inline %s", fun.Def.Name.Raw)
	cg.pushIndent()

	if len(fun.Def.Params) > 0 {
		params := make([]string, len(fun.Def.Params))
		for i, param := range fun.Def.Params {
			params[i] = param.Name.Raw
		}
		cg.ln("local %s = %s;", strings.Join(params, ", "), cg.getCallArgs(call, toCall))
	}

	// Generate return locals
	returnCount := fun.ReturnCount()
	returnLocals := make([]string, returnCount)
	for i := range returnLocals {
		returnLocals[i] = cg.getTempVar()
	}

	bodyResult := cg.genBlockX(fun.Def.Body, BlockNone)

	// Assign body result to return locals
	cg.ln("%s = %s;", strings.Join(returnLocals, ", "), bodyResult)

	cg.popIndent()
	cg.ln("end")

	return strings.Join(returnLocals, ", ")
}

func (cg *Codegen) genCall(call *ast.Call, toCall string, toCallTy ast.SemType) string {
	var fun ast.SemFunction
	if call.Method != nil {
		st := toCallTy.Struct()
		fun = st.Methods[call.Method.Raw]
	} else {
		fun = toCallTy.Function()
	}

	var canInline = func() bool {
		if fun.HasVarargParam() || fun.HasVarargReturn() {
			return false
		}
		hasInlineAttr := false
		for _, attr := range fun.Def.Attributes {
			if attr.Key.Raw == "inline" {
				hasInlineAttr = true
				break
			}
		}
		if !hasInlineAttr {
			return false
		}
		return true
	}

	if canInline() {
		return cg.genInlineCall(call, fun, toCall)
	}

	var callExpr string
	args := cg.getCallArgs(call, toCall)
	if call.Method != nil {
		st := toCallTy.Struct()
		stName := cg.decorateStName(st)
		callExpr = fmt.Sprintf("%s.%s(%s)", stName, call.Method.Raw, args)
	} else {
		callExpr = fmt.Sprintf("%s(%s)", toCall, args)
	}
	if fun.HasVarargReturn() {
		return callExpr
	}
	if !call.IsTryCall && call.Catch == nil {
		return callExpr
	}
	locals := make([]string, fun.ReturnCount())
	for i := range locals {
		locals[i] = cg.getTempVar()
	}
	errorTemp := cg.getTempVar()
	cg.ln("do")
	cg.pushIndent()
	cg.ln("%s, %s = %s;", errorTemp, strings.Join(locals, ", "), callExpr)
	cg.ln("if %s ~= nil then", errorTemp)
	cg.pushIndent()
	if call.IsTryCall {
		cg.ln("return %s;", errorTemp)
	} else {
		catch := call.Catch
		cg.ln("local %s = %s;", catch.Name.Raw, errorTemp)
		blockExpr := ast.NewExpr(&catch.Block)
		cg.ln("%s = %s;", strings.Join(locals, ", "), cg.genExprX(blockExpr))
	}
	cg.popIndent()
	cg.ln("end")
	cg.popIndent()
	cg.ln("end")
	return strings.Join(locals, ", ")
}

func (cg *Codegen) genMethodCall(call *ast.Call, toCall string, toCallTy ast.SemType) string {
	return cg.genCall(call, toCall, toCallTy)
}
