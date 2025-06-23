package lexer

import (
	"strings"

	"github.com/gluax-lang/gluax/common"
)

// TokComment represents a comment token.
type TokComment struct {
	Text      string
	span      common.Span
	Multiline bool
}

func (t TokComment) isToken() {}

func (t TokComment) Span() common.Span {
	return t.span
}

func (t TokComment) String() string {
	return t.Text
}

func (t TokComment) Is(_ string) bool {
	return false
}

func (t TokComment) AsString() string {
	return ""
}

/* Lexing */

func (lx *lexer) comment() *TokComment {
	// sb will hold the comment text (excluding the leading `//`)
	var sb strings.Builder

	lx.Advance() // skip '/'
	lx.Advance() // skip '/'

	// read until newline or EOF
	for c := lx.CurChr; c != nil && *c != '\n'; c = lx.CurChr {
		sb.WriteRune(*c)
		lx.Advance()
	}

	return &TokComment{
		Text:      sb.String(),
		span:      lx.CurrentSpan(),
		Multiline: false,
	}
}

func (lx *lexer) multilineComment() (Token, *diagnostic) {
	var sb strings.Builder

	// Skip the initial "/*"
	lx.Advance() // '/'
	lx.Advance() // '*'

	for c := lx.CurChr; c != nil; c = lx.CurChr {
		// Check if we've reached "*/".
		if *c == '*' {
			if p := lx.Peek(); p != nil && *p == '/' {
				// Consume '*' and '/'
				lx.Advance()
				lx.Advance()

				// Create the comment token and return.
				return &TokComment{
					Text:      sb.String(),
					span:      lx.CurrentSpan(),
					Multiline: true,
				}, nil
			}
		}

		// Otherwise accumulate the current character into the comment text.
		sb.WriteRune(*c)
		lx.Advance()
	}

	// If we get here, we ran out of characters (EOF) without finding "*/".
	return nil, lx.Error("unterminated multiline comment")
}
