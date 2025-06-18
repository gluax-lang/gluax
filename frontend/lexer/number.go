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
	if !isAsciiDigit(lx.curChr) {
		return nil, nil
	}

	isDecimalDigit := func(r rune) bool { return '0' <= r && r <= '9' }
	isHexDigit := func(r rune) bool { return isDecimalDigit(r) || ('a' <= r && r <= 'f') || ('A' <= r && r <= 'F') }

	var sb strings.Builder

	// hexadecimal
	if isChr(lx.curChr, '0') {
		if p := lx.peek(); p != nil && (*p == 'x' || *p == 'X') {
			sb.WriteRune('0')
			lx.advance()             // Consume '0'
			sb.WriteRune(*lx.curChr) // Write 'x' or 'X'
			lx.advance()             // Consume 'x' or 'X'

			if !lx.scanDigits(&sb, isHexDigit) {
				return nil, lx.error("missing hexadecimal digits after '0x'")
			}
			return newTokNumber(sb.String(), lx.currentSpan()), nil
		}
	}

	lx.scanDigits(&sb, isDecimalDigit)

	// fractional part
	if isChr(lx.curChr, '.') {
		sb.WriteRune('.')
		lx.advance() // Consume '.'
		lx.scanDigits(&sb, isDecimalDigit)
	}

	// Exponent part: [eE][+-]?<digits>
	if isChr(lx.curChr, 'e') || isChr(lx.curChr, 'E') {
		sb.WriteRune(*lx.curChr) // Consume 'e' or 'E'
		lx.advance()

		// Optional sign
		if isChr(lx.curChr, '+') || isChr(lx.curChr, '-') {
			sb.WriteRune(*lx.curChr)
			lx.advance()
		}

		if !lx.scanDigits(&sb, isDecimalDigit) {
			return nil, lx.error("missing exponent digits")
		}
	}

	return newTokNumber(sb.String(), lx.currentSpan()), nil
}

func isAsciiDigit(c *rune) bool {
	return c != nil && '0' <= *c && *c <= '9'
}

func (lx *lexer) scanDigits(sb *strings.Builder, isDigit func(r rune) bool) (hasDigits bool) {
	for c := lx.curChr; c != nil; c = lx.curChr {
		if isDigit(*c) {
			hasDigits = true
			sb.WriteRune(*c)
		} else if *c != '_' {
			// Not a digit and not an underscore, so we're done.
			break
		}
		// If it was a digit or an underscore, we advance.
		lx.advance()
	}
	return hasDigits
}
