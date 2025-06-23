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
		lx.Advance()
		return newTokPunct(PunctPlus, lx.CurrentSpan())
	case '-':
		lx.Advance()
		if IsChr(lx.CurChr, '>') {
			lx.Advance()
			return newTokPunct(PunctArrow, lx.CurrentSpan())
		}
		return newTokPunct(PunctMinus, lx.CurrentSpan())
	case '*':
		lx.Advance()
		if IsChr(lx.CurChr, '*') {
			lx.Advance()
			return newTokPunct(PunctExponent, lx.CurrentSpan())
		}
		return newTokPunct(PunctAsterisk, lx.CurrentSpan())
	case '/':
		lx.Advance()
		return newTokPunct(PunctSlash, lx.CurrentSpan())
	case '%':
		lx.Advance()
		return newTokPunct(PunctPercent, lx.CurrentSpan())
	case '=':
		lx.Advance()
		if IsChr(lx.CurChr, '=') {
			lx.Advance()
			return newTokPunct(PunctEqualEqual, lx.CurrentSpan())
		}
		return newTokPunct(PunctEqual, lx.CurrentSpan())
	case '!':
		lx.Advance()
		if IsChr(lx.CurChr, '=') {
			lx.Advance()
			return newTokPunct(PunctNotEqual, lx.CurrentSpan())
		}
		return newTokPunct(PunctBang, lx.CurrentSpan())
	case '<':
		lx.Advance()
		if IsChr(lx.CurChr, '=') {
			lx.Advance()
			return newTokPunct(PunctLessThanEqual, lx.CurrentSpan())
		}
		return newTokPunct(PunctLessThan, lx.CurrentSpan())
	case '>':
		lx.Advance()
		if IsChr(lx.CurChr, '=') {
			lx.Advance()
			return newTokPunct(PunctGreaterThanEqual, lx.CurrentSpan())
		}
		return newTokPunct(PunctGreaterThan, lx.CurrentSpan())
	case '^':
		lx.Advance()
		return newTokPunct(PunctCaret, lx.CurrentSpan())
	case '.':
		lx.Advance()
		if IsChr(lx.CurChr, '.') {
			lx.Advance()
			if IsChr(lx.CurChr, '.') {
				lx.Advance()
				return newTokPunct(PunctVararg, lx.CurrentSpan())
			}
			return newTokPunct(PunctConcat, lx.CurrentSpan())
		}
		return newTokPunct(PunctDot, lx.CurrentSpan())
	case '#':
		lx.Advance()
		return newTokPunct(PunctHash, lx.CurrentSpan())
	case '&':
		lx.Advance()
		if IsChr(lx.CurChr, '&') {
			lx.Advance()
			return newTokPunct(PunctAndAnd, lx.CurrentSpan())
		}
		return newTokPunct(PunctAmpersand, lx.CurrentSpan())
	case '|':
		lx.Advance()
		if IsChr(lx.CurChr, '|') {
			lx.Advance()
			return newTokPunct(PunctOrOr, lx.CurrentSpan())
		}
		return newTokPunct(PunctPipe, lx.CurrentSpan())
	case ';':
		lx.Advance()
		return newTokPunct(PunctSemicolon, lx.CurrentSpan())
	case ':':
		lx.Advance()
		if IsChr(lx.CurChr, ':') {
			lx.Advance()
			return newTokPunct(PunctDoubleColon, lx.CurrentSpan())
		}
		return newTokPunct(PunctColon, lx.CurrentSpan())
	case ',':
		lx.Advance()
		return newTokPunct(PunctComma, lx.CurrentSpan())
	case '(':
		lx.Advance()
		return newTokPunct(PunctOpenParen, lx.CurrentSpan())
	case ')':
		lx.Advance()
		return newTokPunct(PunctCloseParen, lx.CurrentSpan())
	case '{':
		lx.Advance()
		return newTokPunct(PunctOpenBrace, lx.CurrentSpan())
	case '}':
		lx.Advance()
		return newTokPunct(PunctCloseBrace, lx.CurrentSpan())
	case '[':
		lx.Advance()
		return newTokPunct(PunctOpenBracket, lx.CurrentSpan())
	case ']':
		lx.Advance()
		return newTokPunct(PunctCloseBracket, lx.CurrentSpan())
	case '?':
		lx.Advance()
		return newTokPunct(PunctQuestion, lx.CurrentSpan())
	case '@':
		lx.Advance()
		return newTokPunct(PunctAt, lx.CurrentSpan())
	case '~':
		lx.Advance()
		return newTokPunct(PunctTilde, lx.CurrentSpan())
	default:
		return nil
	}
}
