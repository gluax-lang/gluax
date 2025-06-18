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

	if !isIdentStart(*lx.curChr) {
		return nil, lx.error(fmt.Sprintf("unexpected character: %c", *lx.curChr))
	}

	sb.WriteRune(*lx.curChr)
	lx.advance()

	for c := lx.curChr; c != nil && isIdentContinue(*c); c = lx.curChr {
		sb.WriteRune(*c)
		lx.advance()
	}

	if strings.HasPrefix(sb.String(), frontend.PreservedPrefix) {
		return nil, lx.error(fmt.Sprintf("cannot have identifier starting with %s", frontend.PreservedPrefix))
	}

	return NewTokIdent(sb.String(), lx.currentSpan()), nil
}

func isIdentStart(r rune) bool {
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

func isIdentContinue(r rune) bool {
	// Digits are allowed after the first rune.
	if '0' <= r && r <= '9' {
		return true
	}
	// Otherwise the same rules as the first rune.
	return isIdentStart(r)
}

func IsValidIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !isIdentStart(r) {
				return false
			}
		} else {
			if !isIdentContinue(r) {
				return false
			}
		}
	}
	return true
}
