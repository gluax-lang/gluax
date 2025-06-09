package parser

import (
	"fmt"

	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/common"
	"github.com/gluax-lang/gluax/frontend/lexer"
	protocol "github.com/gluax-lang/lsp"
)

type diagnostic = protocol.Diagnostic

type Span = common.Span

var SpanFrom = common.SpanFrom

func errorToDiagnostic(err any) *diagnostic {
	switch err := err.(type) {
	case *diagnostic:
		return err
	default:
		panic(fmt.Errorf("unexpected error: %v", err))
	}
}

type parser struct {
	TokenStream       []lexer.Token
	Token             lexer.Token
	Pos               uint32
	processingGlobals bool
}

func Parse(tkS []lexer.Token, processingGlobals bool) (astRet *ast.Ast, err *diagnostic) {
	p := &parser{
		TokenStream:       tkS,
		Token:             tkS[0],
		Pos:               0,
		processingGlobals: processingGlobals,
	}

	defer func() {
		if r := recover(); r != nil {
			err = errorToDiagnostic(r)
		}
	}()

	astRet = &ast.Ast{
		TokenStream: p.TokenStream,
	}

	const (
		sectionImports = iota
		sectionUses
		sectionOther
	)
	section := sectionImports

	for !lexer.IsEOF(p.Token) {
		item := p.parseItem()
		switch item := item.(type) {
		case *ast.Import:
			if section != sectionImports {
				common.PanicDiag("import statements must appear before any other items", item.Span())
			}
			astRet.Imports = append(astRet.Imports, item)
		case *ast.Use:
			if section == sectionOther {
				common.PanicDiag(
					`"use" statements must appear before any item (only comments and imports may precede them).`,
					item.Span(),
				)
			}
			section = sectionUses
			astRet.Uses = append(astRet.Uses, item)
		default:
			section = sectionOther
			switch item := item.(type) {
			case *ast.Function:
				astRet.Funcs = append(astRet.Funcs, item)
			case *ast.ImplStruct:
				astRet.ImplStructs = append(astRet.ImplStructs, item)
			case *ast.ImplTraitForStruct:
				astRet.ImplTraits = append(astRet.ImplTraits, item)
			case *ast.Let:
				astRet.Lets = append(astRet.Lets, item)
			case *ast.Struct:
				astRet.Structs = append(astRet.Structs, item)
			case *ast.Trait:
				astRet.Traits = append(astRet.Traits, item)
			}
		}
	}

	return
}

// advance moves the parser forward by one token.
func (p *parser) advance() {
	p.Pos = common.MinUint32(p.Pos+1, uint32(len(p.TokenStream)-1))
	p.Token = p.TokenStream[p.Pos]
}

func (p *parser) peek() lexer.Token {
	return p.peekOffset(+1)
}

func (p *parser) peekN(n int) lexer.Token {
	return p.peekOffset(n)
}

func (p *parser) tryConsume(punct string) bool {
	if p.Token.Is(punct) {
		p.advance()
		return true
	}
	return false
}

func (p *parser) expect(s string) {
	if !p.tryConsume(s) {
		common.PanicDiag(fmt.Sprintf("expected: %s", s), p.span())
	}
}

func (p *parser) expectString() lexer.TokString {
	if s, ok := p.Token.(lexer.TokString); ok {
		p.advance()
		return s
	}
	common.PanicDiag("expected string literal", p.span())
	panic("unreachable") // love go
}

func (p *parser) expectNumber() lexer.TokNumber {
	if n, ok := p.Token.(lexer.TokNumber); ok {
		p.advance()
		return n
	}
	common.PanicDiag("expected number literal", p.span())
	panic("unreachable") // love go
}

func (p *parser) expectIdentMsgX(msg string, flags Flags) lexer.TokIdent {
	tok := p.Token
	if i, ok := tok.(lexer.TokIdent); ok {
		p.advance()
		return i
	}
	if flags.Has(FlagAllowUnderscore) && tok.Is("_") {
		p.advance()
		return lexer.NewTokIdent("_", tok.Span())
	}
	common.PanicDiag(fmt.Sprintf("%s, got: %s", msg, tok.String()), tok.Span())
	panic("unreachable") // love go
}

func (p *parser) expectIdentMsg(msg string) lexer.TokIdent {
	return p.expectIdentMsgX(msg, 0)
}

func (p *parser) expectIdent() lexer.TokIdent {
	return p.expectIdentMsg("expected identifier")
}

// peekOffset returns the token at p.Pos + n, clamped to [0, len-1].
// Negative n looks backwards, positive n looks ahead.
func (p *parser) peekOffset(n int) lexer.Token {
	// compute target index as an int
	idx := int(p.Pos) + n

	// clamp to [0, lastIndex]
	if idx < 0 {
		idx = 0
	} else if idx >= len(p.TokenStream) {
		idx = len(p.TokenStream) - 1
	}

	return p.TokenStream[idx]
}

func (p *parser) spanN(n int) common.Span {
	return p.peekOffset(n).Span()
}

func (p *parser) span() common.Span {
	return p.spanN(0)
}

func (p *parser) prevSpan() common.Span {
	return p.spanN(-1)
}

func (p *parser) parseCommaSeparatedDelimited(
	closing string,
	parse func(*parser),
) {
	for !p.Token.Is(closing) {
		parse(p)
		if !p.tryConsume(",") {
			break
		}
	}
	p.expect(closing)
}
