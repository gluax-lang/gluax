package lexer

import (
	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend/lexer/peekable"
	protocol "github.com/gluax-lang/lsp"
)

type diagnostic = protocol.Diagnostic

// lexer is a hand-rolled, rune-based scanner.
type lexer struct {
	src                           string // source is the file being scanned
	Chars                         *peekable.Chars
	CurChr                        *rune
	Line, Column                  uint32
	SavedLine, SavedColumn        uint32
	ColumnUTF16, SavedColumnUTF16 uint32 // for LSP, which uses UTF-16 code units
}

func Lex(src, code string) ([]Token, *diagnostic) {
	var tokens []Token
	lx := NewLexer(src, code)
	for {
		tok, err := lx.NextToken()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if _, ok := tok.(TokEOF); ok {
			break
		}
	}
	return tokens, nil
}

// NewLexer returns a fresh lexer initialised with src.
func NewLexer(src, code string) *lexer {
	chars := peekable.NewPeekableChars(code)
	lx := &lexer{
		src:    src,
		Chars:  chars,
		CurChr: chars.Next(),
		Line:   0, Column: 0,
		SavedLine: 0, SavedColumn: 0,
		ColumnUTF16: 0, SavedColumnUTF16: 0,
	}
	return lx
}

func (lx *lexer) CurrentSpan() common.Span {
	span := common.SpanNew(lx.SavedLine, lx.Line, lx.SavedColumn, lx.Column, lx.SavedColumnUTF16, lx.ColumnUTF16)
	span.Source = lx.src
	return span
}

func (lx *lexer) Advance() {
	c := lx.CurChr
	if c != nil {
		if *c == '\n' {
			lx.Line++
			lx.Column = 0
			lx.ColumnUTF16 = 0
		} else {
			lx.Column++
			if *c > 0xFFFF {
				lx.ColumnUTF16 += 2 // It's a surrogate pair in UTF-16
			} else {
				lx.ColumnUTF16++
			}
		}
	}
	lx.CurChr = lx.Chars.Next()
}

func (lx *lexer) Peek() *rune {
	return lx.Chars.Peek()
}

func (lx *lexer) Error(msg string) *diagnostic {
	return common.ErrorDiag(msg, lx.CurrentSpan())
}

// SkipWs skips whitespaces to the next non-whitespace character.
func (lx *lexer) SkipWs() {
	for {
		c := lx.CurChr
		if !IsWsChr(c) {
			break
		}
		lx.Advance() // skip
	}
	lx.SavedLine = lx.Line
	lx.SavedColumn = lx.Column
	lx.SavedColumnUTF16 = lx.ColumnUTF16
}

func (lx *lexer) NextToken() (Token, *diagnostic) {
	lastLine, lastColumn := lx.Line, lx.Column
	lastColumnUTF16 := lx.ColumnUTF16
	lx.SkipWs() // skip whitespaces

	c := lx.CurChr

	// EOF
	if c == nil {
		span := common.SpanNew(lastLine, lastLine, lastColumn, lastColumn, lastColumnUTF16, lastColumnUTF16)
		span.Source = lx.src
		return TokEOF{span: span}, nil
	}

	if *c == 'r' {
		if p := lx.Peek(); p != nil && (*p == '"' || *p == '#') {
			return lx.rawString()
		}
	}

	// Comment
	if *c == '/' {
		if pC := lx.Peek(); pC != nil {
			switch *pC {
			case '/':
				comment := lx.comment()
				return lx.NextToken() // just for now
				return comment, nil
			case '*':
				comment, dig := lx.multilineComment()
				if dig != nil {
					return nil, dig
				}
				return lx.NextToken() // just for now
				return comment, nil
			}
		}
	}

	// Punctuation
	if token := lx.punct(c); token != nil {
		return token, nil
	}

	// String
	if token, err := lx.string(); err != nil {
		return nil, err
	} else if token != nil {
		return token, nil
	}

	// Number
	if token, err := lx.number(); err != nil {
		return nil, err
	} else if token != nil {
		return token, nil
	}

	// Identifier
	identTok, err := lx.identifier()
	if err != nil {
		return nil, err
	}

	// Keyword
	if keyword, ok := lookupKeyword(identTok.(TokIdent).Raw); ok {
		return newTokKeyword(keyword, identTok.Span()), nil
	}

	return identTok, nil
}

func IsChr(c *rune, e rune) bool {
	return c != nil && *c == e
}

func IsWsChr(c *rune) bool {
	if c == nil {
		return false
	}
	switch *c {
	case ' ', '\t', '\n':
		return true
	default:
		return false
	}
}
