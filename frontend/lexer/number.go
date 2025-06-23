package lexer

import (
	"strings"

	"github.com/gluax-lang/gluax/common"
)

type TokNumber struct {
	Raw  string
	span common.Span
}

func (t TokNumber) isToken() {}

func (t TokNumber) Span() common.Span {
	return t.span
}

func (t TokNumber) String() string {
	return t.Raw
}

func (t TokNumber) Is(_ string) bool {
	return false
}

func (t TokNumber) AsString() string {
	return ""
}

func newTokNumber(s string, span common.Span) TokNumber {
	return TokNumber{Raw: s, span: span}
}

/* Lexing */

func (lx *lexer) number() (Token, *diagnostic) {
	// if does not start with a digit, then return nil
	if !isAsciiDigit(lx.CurChr) {
		return nil, nil
	}

	isDecimalDigit := func(r rune) bool { return '0' <= r && r <= '9' }
	isHexDigit := func(r rune) bool { return isDecimalDigit(r) || ('a' <= r && r <= 'f') || ('A' <= r && r <= 'F') }

	var sb strings.Builder

	// hexadecimal
	if IsChr(lx.CurChr, '0') {
		if p := lx.Peek(); p != nil && (*p == 'x' || *p == 'X') {
			sb.WriteRune('0')
			lx.Advance()             // Consume '0'
			sb.WriteRune(*lx.CurChr) // Write 'x' or 'X'
			lx.Advance()             // Consume 'x' or 'X'

			if !lx.scanDigits(&sb, isHexDigit) {
				return nil, lx.Error("missing hexadecimal digits after '0x'")
			}
			return newTokNumber(sb.String(), lx.CurrentSpan()), nil
		}
	}

	lx.scanDigits(&sb, isDecimalDigit)

	// fractional part
	if IsChr(lx.CurChr, '.') {
		sb.WriteRune('.')
		lx.Advance() // Consume '.'
		lx.scanDigits(&sb, isDecimalDigit)
	}

	// Exponent part: [eE][+-]?<digits>
	if IsChr(lx.CurChr, 'e') || IsChr(lx.CurChr, 'E') {
		sb.WriteRune(*lx.CurChr) // Consume 'e' or 'E'
		lx.Advance()

		// Optional sign
		if IsChr(lx.CurChr, '+') || IsChr(lx.CurChr, '-') {
			sb.WriteRune(*lx.CurChr)
			lx.Advance()
		}

		if !lx.scanDigits(&sb, isDecimalDigit) {
			return nil, lx.Error("missing exponent digits")
		}
	}

	return newTokNumber(sb.String(), lx.CurrentSpan()), nil
}

func isAsciiDigit(c *rune) bool {
	return c != nil && '0' <= *c && *c <= '9'
}

func (lx *lexer) scanDigits(sb *strings.Builder, isDigit func(r rune) bool) (hasDigits bool) {
	for c := lx.CurChr; c != nil; c = lx.CurChr {
		if isDigit(*c) {
			hasDigits = true
			sb.WriteRune(*c)
		} else if *c != '_' {
			// Not a digit and not an underscore, so we're done.
			break
		}
		// If it was a digit or an underscore, we advance.
		lx.Advance()
	}
	return hasDigits
}
