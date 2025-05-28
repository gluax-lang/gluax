package parser

import (
	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/common"
)

func (p *parser) parseItem() ast.Item {
	var attributes []ast.Attribute
	for p.Token.Is("#") {
		attributes = append(attributes, p.parseAttribute())
	}
	public := p.tryConsume("pub")
	var item ast.Item
	switch p.Token.AsString() {
	case "let":
		item = p.parseLet(true)
	case "struct":
		item = p.parseStruct()
	case "use":
		item = p.parseUse()
	case "import":
		item = p.parseImport()
	case "func":
		item = p.parseFunction()
	default:
		common.PanicDiag("expected item", p.span())
	}
	ast.SetItemPublic(item, public)
	if len(attributes) > 0 {
		if !ast.SetItemAttributes(item, attributes) {
			common.PanicDiag("cannot set attributes on item", item.Span())
		}
	}
	return item
}

func (p *parser) parseFunction() ast.Item {
	spanStart := p.span()
	p.advance() // skip `func`
	name := p.expectIdentMsg("expected function name")
	sig := p.parseFunctionSignature(FlagFuncParamVarArg | FlagFuncParamNamed)
	body := p.parseBlock()
	span := SpanFrom(spanStart, p.prevSpan())
	fun := ast.NewFunction(&name, sig, &body, nil, span)
	fun.IsGlobalDef = p.processingGlobals
	return fun
}

func (p *parser) parseStruct() ast.Item {
	spanStart := p.span()

	p.expect("struct")

	name := p.expectIdentMsg("expected struct name")
	generics := p.parseGenerics()

	p.expect("{")

	var (
		fields     []ast.StructField
		methods    []ast.Function
		seenMethod bool
	)

	fieldId := 1 // Start field IDs at 1

	for !p.Token.Is("}") {
		var attributes []ast.Attribute
		for p.Token.Is("#") {
			attributes = append(attributes, p.parseAttribute())
		}
		if p.Token.Is("func") {
			seenMethod = true
			method := p.parseStructMethod()
			method.Attributes = attributes
			methods = append(methods, method)
		} else {
			if seenMethod {
				common.PanicDiag("cannot define fields after methods", p.span())
			}

			fields = append(fields, p.parseStructField(fieldId))
			fieldId++ // Increment field ID for the next field

			// optional trailing comma
			if !p.tryConsume(",") {
				break
			}
		}
	}
	p.expect("}")

	span := SpanFrom(spanStart, p.prevSpan())

	return ast.NewStruct(name, generics, fields, methods, span)
}

func (p *parser) parseStructField(id int) ast.StructField {
	public := p.tryConsume("pub")
	name := p.expectIdent()
	p.expect(":")
	ty := p.parseType()
	return ast.StructField{
		Id:     id,
		Name:   name,
		Type:   ty,
		Public: public,
	}
}

func (p *parser) parseStructMethod() ast.Function {
	spanStart := p.span()

	p.expect("func")
	name := p.expectIdent()

	sig := p.parseFunctionSignature(
		FlagFuncParamVarArg |
			FlagFuncParamSelf |
			FlagFuncParamNamed,
	)

	body := p.parseBlock()

	span := SpanFrom(spanStart, p.prevSpan())

	return *ast.NewFunction(&name, sig, &body, nil, span)
}

func (p *parser) parseImport() ast.Item {
	spanStart := p.span()
	p.advance() // skip `import`

	path := p.expectString()

	var as *ast.Ident
	if p.tryConsume("as") {
		ident := p.expectIdent()
		as = &ident
	}

	p.expect(";")

	span := SpanFrom(spanStart, p.prevSpan())

	return ast.NewImport(path, as, span)
}

func (p *parser) parseUse() ast.Item {
	spanStart := p.span()
	p.advance() // skip `use`

	path := p.parsePath()

	var as *ast.Ident
	if p.tryConsume("as") {
		pAs := p.expectIdent()
		as = &pAs
	}

	p.expect(";")

	span := SpanFrom(spanStart, p.prevSpan())

	use := ast.NewUse(path, as, span)
	return &use
}
