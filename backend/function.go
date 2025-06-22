package codegen

import (
	"fmt"
	"strings"

	"github.com/gluax-lang/gluax/frontend"
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (cg *Codegen) decorateFuncName(f *ast.SemFunction) string {
	if f.IsGlobal() {
		return f.GlobalName()
	}
	raw := f.Def.Name.Raw
	if f.Trait != nil {
		dTName := cg.decorateTraitName(f.Trait.Def, f.Class)
		return dTName + "." + raw
	}
	if f.Class != nil {
		stName := cg.decorateClassName(f.Class)
		return stName + "." + raw
	}
	var sb strings.Builder
	sb.WriteString(frontend.FUNC_PREFIX)
	sb.WriteString(raw)
	if f.Def.IsItem {
		id := fmt.Sprintf("_%d", f.Def.Span().ID)
		sb.WriteString(id)
	}
	baseName := sb.String()
	if f.Def.IsItem {
		return cg.getPublic(baseName) + fmt.Sprintf(" --[[%s]]", f.String())
	}
	return baseName + fmt.Sprintf(" --[[%s]]", f.String())
}

func (cg *Codegen) genFunctionParams(f ast.Function) []string {
	params := make([]string, len(f.Params))
	for i, p := range f.Params {
		params[i] = p.String()
	}
	return params
}

func (cg *Codegen) genFunction(f *ast.SemFunction) string {
	if f.IsGlobal() {
		return f.GlobalName()
	}
	def := f.Def
	oldBuf := cg.newBuf()

	// Generate function signature
	cg.writeString("function(")
	cg.writeString(strings.Join(cg.genFunctionParams(f.Def), ", "))
	cg.writeByte(')')
	cg.writeByte('\n')
	cg.pushIndent()

	// Setup scopes for function body
	cg.pushTempScope()
	cg.pushFuncScope(&funcScope{})

	// Prepare buffer for function body
	bodyBuf := cg.newBuf()

	// Generate function body and return statement
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

	cg.popFuncScope()

	// Emit locals and body
	bodySnippet := cg.restoreBuf(bodyBuf)
	cg.emitTempLocals()
	cg.writeString(bodySnippet)

	// Close function
	cg.popIndent()
	cg.writeIndent()
	cg.writeString("end")

	// Restore and return the generated function code
	return cg.restoreBuf(oldBuf)
}

func (cg *Codegen) getCallArgs(call *ast.Call, toCall string) string {
	args := cg.genExprsLeftToRight(call.Args)
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
	cg.ln("do --[[inline call: %s]]", fun.Def.Name.Raw)
	cg.pushIndent()

	if len(fun.Def.Params) > 0 {
		params := make([]string, len(fun.Def.Params))
		for i, param := range fun.Def.Params {
			params[i] = param.Name.Raw
		}
		cg.ln("local %s = %s;", strings.Join(params, ", "), cg.getCallArgs(call, toCall))
	}

	returnLabel := cg.namedTemp(frontend.RETURN_PREFIX)

	// Generate return locals
	returnCount := fun.ReturnCount()
	returnLocals := make([]string, returnCount)
	for i := range returnLocals {
		returnLocals[i] = cg.getTempVar()
	}

	var errTemp string
	if fun.Def.Errorable {
		errTemp = cg.getTempVar()
		cg.ln("%s = nil;", errTemp) // make sure that if it's a reused temp, it starts as nil
	}

	funcScope := funcScope{
		inlining:    true,
		returnLabel: returnLabel,
		returnVars:  returnLocals,
		errorVar:    errTemp,
	}

	cg.pushFuncScope(&funcScope)
	bodyResult := cg.genBlockX(fun.Def.Body, BlockNone)
	cg.popFuncScope()

	cg.ln("%s = %s;", strings.Join(returnLocals, ", "), bodyResult)

	if funcScope.usedLabel {
		cg.ln("::%s::", returnLabel)
	}

	cg.popIndent()
	cg.ln("end")

	if fun.Def.Errorable {
		return errTemp + ", " + strings.Join(returnLocals, ", ")
	}
	return strings.Join(returnLocals, ", ")
}

func (cg *Codegen) genCall(call *ast.Call, toCall string, toCallTy ast.SemType) string {
	fun := call.SemaFunc

	if fun.Def.Attributes.Has("no_op") {
		// If the function has a "no_op" attribute, we don't generate any code for it.
		return "nil"
	}

	canInline := func() bool {
		if fun.HasVarargParam() || fun.HasVarargReturn() {
			return false
		}
		if fun.IsGlobal() {
			return false
		}
		if fun.Def.Body == nil {
			return false
		}
		if !fun.Def.Attributes.Has("inline") {
			return false
		}
		return true
	}

	buildCallExpr := func() string {
		if call.Method != nil {
			switch {
			case toCallTy.IsClass():
				if fun.Trait != nil {
					args := cg.getCallArgs(call, toCall)
					return fmt.Sprintf("%s(%s)", cg.decorateFuncName(fun), args)
				} else if toCallTy.IsClass() && toCallTy.Class().Def.Attributes.Has("no_metatable", "no__index") {
					args := cg.getCallArgs(call, toCall)
					return fmt.Sprintf("%s(%s)", cg.decorateFuncName(fun), args)
				} else {
					args := cg.genExprsLeftToRight(call.Args)
					var name string
					if rename := fun.Attributes().GetString("method_rename"); rename != nil {
						name = *rename
					} else {
						name = fun.Def.Name.Raw
					}
					return fmt.Sprintf("%s:%s(%s)", toCall, name, args)
				}
			case toCallTy.IsDynTrait():
				args := cg.getCallArgs(call, toCall)
				return fmt.Sprintf("%s(%s)", cg.decorateFuncName(fun), args)
			default:
				args := cg.genExprsLeftToRight(call.Args)
				return fmt.Sprintf("%s(%s)", toCall, args)
			}
		}
		args := cg.genExprsLeftToRight(call.Args)
		return fmt.Sprintf("%s(%s)", toCall, args)
	}

	handleErrorable := func(callExpr string) string {
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

	var callExpr string
	if canInline() {
		callExpr = cg.genInlineCall(call, *fun, toCall)
	} else {
		callExpr = buildCallExpr()
	}

	if fun.HasVarargReturn() || (!call.IsTryCall && call.Catch == nil) {
		return callExpr
	}

	return handleErrorable(callExpr)
}
