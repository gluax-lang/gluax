package codegen

import (
	"fmt"
	"strings"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (cg *Codegen) decorateFuncName(f *ast.SemFunction) string {
	raw := f.Def.Name.Raw
	if f.Def.Public && f.Def.IsGlobalDef {
		if rename_to := f.Def.Attributes.GetString("rename_to"); rename_to != nil {
			return *rename_to
		}
		return raw
	}
	if f.Trait != nil {
		dTName := cg.decorateTraitName(f.Trait.Def, f.Class)
		return dTName + "." + f.Def.Name.Raw
	}
	if f.Class != nil {
		stName := cg.decorateClassName(f.Class)
		return stName + "." + f.Def.Name.Raw
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

func (cg *Codegen) genFunctionParams(f ast.Function) []string {
	params := make([]string, len(f.Params))
	for i, p := range f.Params {
		params[i] = p.String()
	}
	return params
}

func (cg *Codegen) genFunction(f *ast.SemFunction) string {
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
	cg.ln("do -- inline %s", fun.Def.Name.Raw)
	cg.pushIndent()

	if len(fun.Def.Params) > 0 {
		params := make([]string, len(fun.Def.Params))
		for i, param := range fun.Def.Params {
			params[i] = param.Name.Raw
		}
		cg.ln("local %s = %s;", strings.Join(params, ", "), cg.getCallArgs(call, toCall))
	}

	returnLabel := cg.namedTemp(RETURN_PREFIX)

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
	var fun ast.SemFunction
	if call.Method != nil {
		switch {
		case toCallTy.IsClass():
			st := toCallTy.Class()
			funP := cg.Analysis.FindClassMethod(st, call.Method.Raw)
			fun = *funP
		case toCallTy.IsDynTrait():
			_ = toCallTy.DynTrait()
			panic("todo")
			// fun, _ = cg.Analysis.GetTraitMethod(dt.Trait, call.Method.Raw)
		}
	} else {
		fun = toCallTy.Function()
	}

	if fun.Def.Attributes.Has("no_op") {
		// If the function has a "no_op" attribute, we don't generate any code for it.
		return "nil"
	}

	canInline := func() bool {
		if fun.HasVarargParam() || fun.HasVarargReturn() {
			return false
		}
		if fun.Def.IsGlobalDef {
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
			if toCallTy.IsClass() && toCallTy.Class().Def.Attributes.Has("no_metatable", "no__index") {
				stName := cg.decorateClassName(toCallTy.Class())
				args := cg.getCallArgs(call, toCall)
				return fmt.Sprintf("%s.%s(%s)", stName, call.Method.Raw, args)
			}
			args := cg.genExprsLeftToRight(call.Args)
			return fmt.Sprintf("%s:%s(%s)", toCall, call.Method.Raw, args)
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
		callExpr = cg.genInlineCall(call, fun, toCall)
	} else {
		callExpr = buildCallExpr()
	}

	if fun.HasVarargReturn() || (!call.IsTryCall && call.Catch == nil) {
		return callExpr
	}

	return handleErrorable(callExpr)
}
