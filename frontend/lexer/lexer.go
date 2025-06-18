package lexer

import (
	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend/lexer/peekable"
	protocol "github.com/gluax-lang/lsp"
)

type diagnostic = protocol.Diagnostic

// lexer is a hand-rolled, rune-based scanner.
type lexer struct {
	src                    string // source is the file being scanned
	chars                  *peekable.Chars
	curChr                 *rune
	line, column           uint32
	savedLine, savedColumn uint32
}

func Lex(src, code string) ([]Token, *diagnostic) {
	var tokens []Token
	lx := newLexer(src, code)
	for {
		tok, err := lx.nextToken()
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

// newLexer returns a fresh lexer initialised with src.
func newLexer(src, code string) *lexer {
	chars := peekable.NewPeekableChars(code)
	lx := &lexer{
		src:    src,
		chars:  chars,
		curChr: chars.Next(),
		line:   1, column: 1,
		savedLine: 1, savedColumn: 1,
	}
	return lx
}

func (lx *lexer) currentSpan() common.Span {
	span := common.SpanNew(lx.savedLine, lx.line, lx.savedColumn, lx.column-1)
	span.Source = lx.src
	return span
}

func (lx *lexer) advance() {
	c := lx.curChr
	if c != nil {
		if *c == '\n' {
			lx.line++
			lx.column = 1
		} else {
			lx.column++
		}
	}
	lx.curChr = lx.chars.Next()
}

func (lx *lexer) peek() *rune {
	return lx.chars.Peek()
}

func (lx *lexer) error(msg string) *diagnostic {
	span := common.SpanNew(lx.savedLine, lx.line, lx.savedColumn, common.MaxUint32(lx.column-1, 1))
	span.Source = lx.src
	return common.ErrorDiag(msg, span)
}

// skipWs skips whitespaces to the next non-whitespace character.
func (lx *lexer) skipWs() {
	for {
		c := lx.curChr
		if !isWsChr(c) {
			break
		}
		lx.advance() // skip
	}
	lx.savedLine = lx.line
	lx.savedColumn = lx.column
}

func (lx *lexer) nextToken() (Token, *diagnostic) {
	lastLine, lastColumn := lx.line, lx.column
	lx.skipWs() // skip whitespaces

	c := lx.curChr

	// EOF
	if c == nil {
		span := common.SpanNew(lastLine, lastLine, lastColumn, lastColumn)
		span.Source = lx.src
		return TokEOF{span: span}, nil
	}

	// Comment
	if *c == '/' {
		if pC := lx.peek(); pC != nil {
			switch *pC {
			case '/':
				comment := lx.comment()
				return lx.nextToken() // just for now
				return comment, nil
			case '*':
				comment, dig := lx.multilineComment()
				if dig != nil {
					return nil, dig
				}
				return lx.nextToken() // just for now
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

func isChr(c *rune, e rune) bool {
	return c != nil && *c == e
}

func isWsChr(c *rune) bool {
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
