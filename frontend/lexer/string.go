package lexer

import (
	"strings"

	"github.com/gluax-lang/gluax/common"
)

// TokString represents a string token.
type TokString struct {
	Raw       string
	span      common.Span
	Multiline bool
}

func (t TokString) isToken() {}

func (t TokString) Span() common.Span {
	return t.span
}

func (t TokString) String() string {
	return t.Raw
}

func (t TokString) Is(_ string) bool {
	return false
}

func (t TokString) AsString() string {
	return ""
}

func NewTokString(s string, span common.Span) TokString {
	return TokString{Raw: s, span: span}
}

/* Lexing */

func (lx *lexer) string() (Token, *diagnostic) {
	// if does not start with a quote, then return nil
	if !isChr(lx.curChr, '"') {
		return nil, nil
	}

	lx.advance() // skip '"'

	var sb strings.Builder

	for {
		c := lx.curChr
		if c == nil {
			// eof, unterminated string
			return nil, common.ErrorDiag("unterminated string literal", lx.currentSpan())
		} else if *c == '"' {
			// end of string
			lx.advance()
			return NewTokString(sb.String(), lx.currentSpan()), nil
		} else if *c == '\\' {
			// escape sequence
			lx.advance()
			if c := lx.curChr; c != nil {
				switch *c {
				case '0', 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', '"', '\'':
					sb.WriteRune('\\')
					sb.WriteRune(*c)
					lx.advance()
				default:
					return nil, common.ErrorDiag("invalid escape sequence", lx.currentSpan())
				}
			} else {
				return nil, common.ErrorDiag("unterminated string literal", lx.currentSpan())
			}
		} else if *c == '\n' {
			return nil, common.ErrorDiag("unterminated string literal", lx.currentSpan())
		} else {
			sb.WriteRune(*c)
			lx.advance()
		}
	}
}
