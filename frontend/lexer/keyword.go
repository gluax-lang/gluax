package lexer

import "github.com/gluax-lang/gluax/common"

// Keyword represents a reserved keyword.
type Keyword int

const (
	_ Keyword = iota
	KwBreak
	KwElse
	KwFalse
	KwFor
	KwFunc
	KwIf
	KwIn
	KwLet
	KwReturn
	KwTrue
	KwWhile
	KwContinue
	KwLoop
	KwImport
	KwSelf
	KwPub
	KwAs
	KwUnsafeCast
	KwClass
	KwUse
	KwThrow
	KwCatch
	KwImpl
	KwTrait
	KwUnreachable
	KwUnderscore
	KwConst
	KwAnd // Lua-reserved below
	KwLocal
	KwDo
	KwElseIf
	KwEnd
	KwFunction
	KwNot
	KwOr
	KwRepeat
	KwUntil
	KwThen
)

// table is populated at compile-time; no code runs in init().
var keywordTable = map[string]Keyword{
	"break":          KwBreak,
	"else":           KwElse,
	"false":          KwFalse,
	"for":            KwFor,
	"func":           KwFunc,
	"if":             KwIf,
	"in":             KwIn,
	"let":            KwLet,
	"return":         KwReturn,
	"true":           KwTrue,
	"while":          KwWhile,
	"continue":       KwContinue,
	"loop":           KwLoop,
	"import":         KwImport,
	"Self":           KwSelf,
	"pub":            KwPub,
	"as":             KwAs,
	"unsafe_cast_as": KwUnsafeCast,
	"class":          KwClass,
	"use":            KwUse,
	"throw":          KwThrow,
	"catch":          KwCatch,
	"impl":           KwImpl,
	"trait":          KwTrait,
	"unreachable":    KwUnreachable,
	"_":              KwUnderscore,
	"const":          KwConst,
	// Lua reserved
	"and":      KwAnd,
	"local":    KwLocal,
	"do":       KwDo,
	"elseif":   KwElseIf,
	"end":      KwEnd,
	"function": KwFunction,
	"not":      KwNot,
	"or":       KwOr,
	"repeat":   KwRepeat,
	"until":    KwUntil,
	"then":     KwThen,
}

var keywordNames = func() []string {
	// find the largest enum value so the slice is the right length
	var max Keyword
	for _, kw := range keywordTable {
		if kw > max {
			max = kw
		}
	}
	names := make([]string, max+1)
	for lit, kw := range keywordTable {
		names[kw] = lit
	}
	return names
}()

func lookupKeyword(lit string) (Keyword, bool) {
	kw, ok := keywordTable[lit]
	return kw, ok
}

type TokKeyword struct {
	Keyword Keyword
	span    common.Span
}

func (t TokKeyword) isToken() {}

func (t TokKeyword) Span() common.Span {
	return t.span
}

func (t TokKeyword) String() string {
	return keywordNames[t.Keyword]
}

func (t TokKeyword) Is(other string) bool {
	return keywordTable[other] == t.Keyword
}

func (t TokKeyword) AsString() string {
	return t.String()
}

func newTokKeyword(k Keyword, span common.Span) TokKeyword {
	return TokKeyword{Keyword: k, span: span}
}
