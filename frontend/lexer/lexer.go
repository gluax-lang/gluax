package lexer

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/gluax-lang/gluax/frontend"
	"github.com/gluax-lang/gluax/frontend/common"
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

func (lx *lexer) punct(c *rune) Token {
	switch *c {
	case '+':
		lx.advance()
		return newTokPunct(PunctPlus, lx.currentSpan())
	case '-':
		lx.advance()
		if isChr(lx.curChr, '>') {
			lx.advance()
			return newTokPunct(PunctArrow, lx.currentSpan())
		}
		return newTokPunct(PunctMinus, lx.currentSpan())
	case '*':
		lx.advance()
		if isChr(lx.curChr, '*') {
			lx.advance()
			return newTokPunct(PunctExponent, lx.currentSpan())
		}
		return newTokPunct(PunctAsterisk, lx.currentSpan())
	case '/':
		lx.advance()
		return newTokPunct(PunctSlash, lx.currentSpan())
	case '%':
		lx.advance()
		return newTokPunct(PunctPercent, lx.currentSpan())
	case '=':
		lx.advance()
		if isChr(lx.curChr, '=') {
			lx.advance()
			return newTokPunct(PunctEqualEqual, lx.currentSpan())
		}
		return newTokPunct(PunctEqual, lx.currentSpan())
	case '!':
		lx.advance()
		if isChr(lx.curChr, '=') {
			lx.advance()
			return newTokPunct(PunctNotEqual, lx.currentSpan())
		}
		return newTokPunct(PunctBang, lx.currentSpan())
	case '<':
		lx.advance()
		if isChr(lx.curChr, '=') {
			lx.advance()
			return newTokPunct(PunctLessThanEqual, lx.currentSpan())
		}
		return newTokPunct(PunctLessThan, lx.currentSpan())
	case '>':
		lx.advance()
		if isChr(lx.curChr, '=') {
			lx.advance()
			return newTokPunct(PunctGreaterThanEqual, lx.currentSpan())
		}
		return newTokPunct(PunctGreaterThan, lx.currentSpan())
	case '^':
		lx.advance()
		return newTokPunct(PunctCaret, lx.currentSpan())
	case '.':
		lx.advance()
		if isChr(lx.curChr, '.') {
			lx.advance()
			if isChr(lx.curChr, '.') {
				lx.advance()
				return newTokPunct(PunctVararg, lx.currentSpan())
			}
			return newTokPunct(PunctConcat, lx.currentSpan())
		}
		return newTokPunct(PunctDot, lx.currentSpan())
	case '#':
		lx.advance()
		return newTokPunct(PunctHash, lx.currentSpan())
	case '&':
		lx.advance()
		if isChr(lx.curChr, '&') {
			lx.advance()
			return newTokPunct(PunctAndAnd, lx.currentSpan())
		}
		return newTokPunct(PunctAmpersand, lx.currentSpan())
	case '|':
		lx.advance()
		if isChr(lx.curChr, '|') {
			lx.advance()
			return newTokPunct(PunctOrOr, lx.currentSpan())
		}
		return newTokPunct(PunctPipe, lx.currentSpan())
	case ';':
		lx.advance()
		return newTokPunct(PunctSemicolon, lx.currentSpan())
	case ':':
		lx.advance()
		if isChr(lx.curChr, ':') {
			lx.advance()
			return newTokPunct(PunctDoubleColon, lx.currentSpan())
		}
		return newTokPunct(PunctColon, lx.currentSpan())
	case ',':
		lx.advance()
		return newTokPunct(PunctComma, lx.currentSpan())
	case '(':
		lx.advance()
		return newTokPunct(PunctOpenParen, lx.currentSpan())
	case ')':
		lx.advance()
		return newTokPunct(PunctCloseParen, lx.currentSpan())
	case '{':
		lx.advance()
		return newTokPunct(PunctOpenBrace, lx.currentSpan())
	case '}':
		lx.advance()
		return newTokPunct(PunctCloseBrace, lx.currentSpan())
	case '[':
		lx.advance()
		return newTokPunct(PunctOpenBracket, lx.currentSpan())
	case ']':
		lx.advance()
		return newTokPunct(PunctCloseBracket, lx.currentSpan())
	case '?':
		lx.advance()
		return newTokPunct(PunctQuestion, lx.currentSpan())
	case '@':
		lx.advance()
		return newTokPunct(PunctAt, lx.currentSpan())

	// Lua 5.1 punctuation
	case '~':
		lx.advance()
		return newTokPunct(PunctTilde, lx.currentSpan())
	default:
		return nil
	}
}

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

func (lx *lexer) number() (Token, *diagnostic) {
	// if does not start with a digit, then return nil
	if !isAsciiDigit(lx.curChr) {
		return nil, nil
	}

	var sb strings.Builder

	// hexadecimal
	if isChr(lx.curChr, '0') {
		sb.WriteRune('0')
		lx.advance()
		if isChr(lx.curChr, 'x') || isChr(lx.curChr, 'X') {
			sb.WriteRune(*lx.curChr)
			lx.advance()
			for c := lx.curChr; c != nil && unicode.Is(unicode.Hex_Digit, *c); c = lx.curChr {
				sb.WriteRune(*c)
				lx.advance()
			}
			return newTokNumber(sb.String(), lx.currentSpan()), nil
		}
	}

	for c := lx.curChr; isAsciiDigit(c); c = lx.curChr {
		sb.WriteRune(*c)
		lx.advance()
	}

	// fractional part
	if isChr(lx.curChr, '.') {
		sb.WriteRune('.') // consume '.'
		lx.advance()

		// The next rune must start a sequence of ASCII digits.
		if c := lx.curChr; isAsciiDigit(c) {
			// consume the run of digits
			for ; isAsciiDigit(c); c = lx.curChr {
				sb.WriteRune(*c)
				lx.advance()
			}
		} else if c != nil { // non-digit encountered immediately after '.'
			return nil, common.ErrorDiag(
				fmt.Sprintf("unexpected character: %c", *c),
				lx.currentSpan(),
			)
		}
	}

	// Exponent part: [eE][+-]?<digits>
	if isChr(lx.curChr, 'e') || isChr(lx.curChr, 'E') {
		sb.WriteRune(*lx.curChr) // consume 'e' or 'E'
		lx.advance()

		// Optional sign
		if isChr(lx.curChr, '+') || isChr(lx.curChr, '-') {
			sb.WriteRune(*lx.curChr)
			lx.advance()
		}

		// Must have at least one digit
		if !isAsciiDigit(lx.curChr) {
			return nil, common.ErrorDiag("missing digit for exponent", lx.currentSpan())
		}

		// Consume the run of digits
		for c := lx.curChr; isAsciiDigit(c); c = lx.curChr {
			sb.WriteRune(*c)
			lx.advance()
		}
	}

	return newTokNumber(sb.String(), lx.currentSpan()), nil
}

func (lx *lexer) comment() *TokComment {
	// sb will hold the comment text (excluding the leading `//`)
	var sb strings.Builder

	lx.advance() // skip '/'
	lx.advance() // skip '/'

	// read until newline or EOF
	for c := lx.curChr; c != nil && *c != '\n'; c = lx.curChr {
		sb.WriteRune(*c)
		lx.advance()
	}

	return &TokComment{
		Text:      sb.String(),
		span:      lx.currentSpan(),
		Multiline: false,
	}
}

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

func (lx *lexer) multilineComment() (Token, *diagnostic) {
	var sb strings.Builder

	// Skip the initial "/*"
	lx.advance() // '/'
	lx.advance() // '*'

	for c := lx.curChr; c != nil; c = lx.curChr {
		// Check if we've reached "*/".
		if *c == '*' {
			if p := lx.peek(); p != nil && *p == '/' {
				// Consume '*' and '/'
				lx.advance()
				lx.advance()

				// Create the comment token and return.
				return &TokComment{
					Text:      sb.String(),
					span:      lx.currentSpan(),
					Multiline: true,
				}, nil
			}
		}

		// Otherwise accumulate the current character into the comment text.
		sb.WriteRune(*c)
		lx.advance()
	}

	// If we get here, we ran out of characters (EOF) without finding "*/".
	return nil, lx.error("unterminated multiline comment")
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

func isAsciiDigit(c *rune) bool {
	return c != nil && '0' <= *c && *c <= '9'
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
