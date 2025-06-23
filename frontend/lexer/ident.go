package lexer

import (
	"fmt"
	"strings"

	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend"
)

type TokIdent struct {
	Raw  string
	span common.Span
}

func (t TokIdent) isToken() {}

func (t TokIdent) Span() common.Span {
	return t.span
}

func (t TokIdent) String() string {
	return t.Raw
}

func (t TokIdent) Is(_ string) bool {
	return false
}

func (t TokIdent) AsString() string {
	return ""
}

func NewTokIdent(s string, span common.Span) TokIdent {
	return TokIdent{Raw: s, span: span}
}

func IsIdentStr(t Token, s string) bool {
	if ident, ok := t.(TokIdent); ok {
		return ident.Raw == s
	}
	return false
}

/* Lexing */

func (lx *lexer) identifier() (Token, *diagnostic) {
	var sb strings.Builder

	if !IsIdentStart(*lx.CurChr) {
		return nil, lx.Error(fmt.Sprintf("unexpected character: %c", *lx.CurChr))
	}

	sb.WriteRune(*lx.CurChr)
	lx.Advance()

	for c := lx.CurChr; c != nil && IsIdentContinue(*c); c = lx.CurChr {
		sb.WriteRune(*c)
		lx.Advance()
	}

	if strings.HasPrefix(sb.String(), frontend.PreservedPrefix) {
		return nil, lx.Error(fmt.Sprintf("cannot have identifier starting with %s", frontend.PreservedPrefix))
	}

	return NewTokIdent(sb.String(), lx.CurrentSpan()), nil
}

func IsIdentStart(r rune) bool {
	// ASCII letters
	if ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z') {
		return true
	}
	// Underscore
	if r == '_' {
		return true
	}
	// Any non-ASCII code-point (LuaJIT treats every byte ≥0x80 as `letter`)
	if r >= 0x80 && r <= 0x10FFFF {
		return true
	}
	return false
}

func IsIdentContinue(r rune) bool {
	// Digits are allowed after the first rune.
	if '0' <= r && r <= '9' {
		return true
	}
	// Otherwise the same rules as the first rune.
	return IsIdentStart(r)
}

func IsValidIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !IsIdentStart(r) {
				return false
			}
		} else {
			if !IsIdentContinue(r) {
				return false
			}
		}
	}
	return true
}

func IsValidIdentRune(r rune) bool {
	return IsIdentStart(r) || IsIdentContinue(r)
}
