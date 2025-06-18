package lexer

import "github.com/gluax-lang/gluax/common"

// Punct represents a punctuation token.
type Punct int

const (
	_ Punct = iota

	// PunctPlus is `+`
	PunctPlus
	// PunctMinus is `-`
	PunctMinus
	// PunctAsterisk is `*`
	PunctAsterisk
	// PunctSlash is `/`
	PunctSlash
	// PunctPercent is `%`
	PunctPercent
	// PunctEqual is `=`
	PunctEqual
	// PunctEqualEqual is `==`
	PunctEqualEqual
	// PunctNotEqual is `!=`
	PunctNotEqual
	// PunctLessThan is `<`
	PunctLessThan
	// PunctLessThanEqual is `<=`
	PunctLessThanEqual
	// PunctGreaterThan is `>`
	PunctGreaterThan
	// PunctGreaterThanEqual is `>=`
	PunctGreaterThanEqual
	// PunctCaret is `^`
	PunctCaret
	// PunctConcat is `..`
	PunctConcat
	// PunctHash is `#`
	PunctHash
	// PunctBang is `!`
	PunctBang
	// PunctAndAnd is `&&`
	PunctAndAnd
	// PunctOrOr is `||`
	PunctOrOr
	// PunctVararg is `...`
	PunctVararg
	// PunctSemicolon is `;`
	PunctSemicolon
	// PunctColon is `:`
	PunctColon
	// PunctDoubleColon is `::`
	PunctDoubleColon
	// PunctComma is `,`
	PunctComma
	// PunctOpenParen is `(`
	PunctOpenParen
	// PunctCloseParen is `)`
	PunctCloseParen
	// PunctOpenBrace is `{`
	PunctOpenBrace
	// PunctCloseBrace is `}`
	PunctCloseBrace
	// PunctOpenBracket is `[`
	PunctOpenBracket
	// PunctCloseBracket is `]`
	PunctCloseBracket
	// PunctDot is `.`
	PunctDot
	// PunctArrow is `->`
	PunctArrow
	// PunctPipe is `|`
	PunctPipe
	// PunctTilde is `~`
	PunctTilde
	// PunctAmpersand is `&`
	PunctAmpersand
	// PunctExponent is `**`
	PunctExponent
	// PunctQuestion is `?`
	PunctQuestion
	// PunctAt is `@`
	PunctAt
)

var puncts = map[string]Punct{
	"+":   PunctPlus,
	"-":   PunctMinus,
	"*":   PunctAsterisk,
	"/":   PunctSlash,
	"%":   PunctPercent,
	"==":  PunctEqualEqual,
	"!=":  PunctNotEqual,
	"<":   PunctLessThan,
	"<=":  PunctLessThanEqual,
	">":   PunctGreaterThan,
	">=":  PunctGreaterThanEqual,
	"=":   PunctEqual,
	"^":   PunctCaret,
	"..":  PunctConcat,
	"#":   PunctHash,
	"!":   PunctBang,
	"&&":  PunctAndAnd,
	"||":  PunctOrOr,
	"...": PunctVararg,
	";":   PunctSemicolon,
	":":   PunctColon,
	"::":  PunctDoubleColon,
	",":   PunctComma,
	"(":   PunctOpenParen,
	")":   PunctCloseParen,
	"{":   PunctOpenBrace,
	"}":   PunctCloseBrace,
	"[":   PunctOpenBracket,
	"]":   PunctCloseBracket,
	".":   PunctDot,
	"->":  PunctArrow,
	"|":   PunctPipe,
	"~":   PunctTilde,
	"&":   PunctAmpersand,
	"**":  PunctExponent,
	"?":   PunctQuestion,
	"@":   PunctAt,
}

var punctNames = func() []string {
	// find the largest enum value so the slice is the right length
	var max Punct
	for _, p := range puncts {
		if p > max {
			max = p
		}
	}
	names := make([]string, max+1)
	for lit, p := range puncts {
		names[p] = lit
	}
	return names
}()

type TokPunct struct {
	Punct Punct
	span  common.Span
}

func (t TokPunct) isToken() {}

func (t TokPunct) Span() common.Span {
	return t.span
}

func (t TokPunct) String() string {
	return punctNames[t.Punct]
}

func (t TokPunct) Is(other string) bool {
	return puncts[other] == t.Punct
}

func (t TokPunct) AsString() string {
	return t.String()
}

func newTokPunct(p Punct, span common.Span) TokPunct {
	return TokPunct{Punct: p, span: span}
}

/* Lexing */

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
