package parser

import (
	"fmt"

	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

// parseAttribute parses Rust-style attributes:
//
//	#[ident]                   -> no input
//	#[ident(token, tree)]      -> token-tree input
//	#[ident = "string"]        -> string input
//
func (p *parser) parseAttribute() ast.Attribute {
	spanStart := p.span() // points at the initial '#'
	p.advance()           // consume '#'

	p.expect("[") // after here we are inside the attribute

	key := p.expectIdent()

	var (
		kind      = ast.AttrInputNone
		tokenTree []lexer.Token
		stringLit *lexer.TokString
	)

	switch {
	// DelimTokenTree
	case p.Token.Is("(") || p.Token.Is("[") || p.Token.Is("{"):
		kind, tokenTree = ast.AttrInputTokenTree, p.parseDelimTokenTree()

	// = "string"
	case p.tryConsume("="):
		str := p.expectString()
		kind, stringLit = ast.AttrInputString, &str
	}

	p.expect("]") // final bracket - end of attribute
	span := SpanFrom(spanStart, p.prevSpan())

	return ast.Attribute{
		Key:       key,
		Kind:      kind,
		TokenTree: tokenTree,
		String:    stringLit,
		Span:      span,
	}
}

// parseDelimTokenTree collects **all** tokens contained in the outer-most
// delimiter - it never interprets them.
func (p *parser) parseDelimTokenTree() []lexer.Token {
	open := p.Token // either '(', '[' or '{'
	spanStart := p.span()
	close := map[string]string{"(": ")", "[": "]", "{": "}"}[open.AsString()]

	var (
		tokens []lexer.Token
		depth  = 0
	)

	// consume opening delimiter
	p.advance()

	for {
		if lexer.IsEOF(p.Token) {
			common.PanicDiag(fmt.Sprintf("unterminated `%s`: expected `%s` before end of input", open, close), spanStart)
		}

		if openTok := p.Token.AsString(); openTok == open.AsString() {
			depth++
		} else if p.Token.Is(close) {
			if depth == 0 {
				break // this is the matching closer for the outer-most open
			}
			depth--
		}

		tokens = append(tokens, p.Token)
		p.advance()
	}

	p.expect(close) // consume the outer-most closing delimiter
	return tokens
}
