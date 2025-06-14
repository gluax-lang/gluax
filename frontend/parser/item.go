package parser

import (
	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend/ast"
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
	case "class":
		item = p.parseClass()
	case "use":
		item = p.parseUse()
	case "import":
		item = p.parseImport()
	case "func":
		item = p.parseFunction()
	case "impl":
		item = p.parseImpl()
	case "trait":
		item = p.parseTrait()
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

func (p *parser) parseClass() ast.Item {
	spanStart := p.span()

	p.advance() // skip `class`

	name := p.expectIdentMsg("expected class name")
	generics := p.parseGenerics()

	var super *ast.Type
	if p.tryConsume(":") {
		t := p.parseType()
		super = &t
	}

	p.expect("{")

	var (
		fields []ast.ClassField
	)

	fieldId := 1 // Start field IDs at 1

	for !p.Token.Is("}") {
		var attributes []ast.Attribute
		for p.Token.Is("#") {
			attributes = append(attributes, p.parseAttribute())
		}

		fields = append(fields, p.parseClassField(fieldId))
		fieldId++ // Increment field ID for the next field

		// optional trailing comma
		if !p.tryConsume(",") {
			break
		}
	}
	p.expect("}")

	span := SpanFrom(spanStart, p.prevSpan())

	st := ast.NewClass(name, generics, super, fields, span)
	st.IsGlobalDef = p.processingGlobals
	return st
}

func (p *parser) parseClassField(id int) ast.ClassField {
	public := p.tryConsume("pub")
	name := p.expectIdent()
	p.expect(":")
	ty := p.parseType()
	return ast.ClassField{
		Id:     id,
		Name:   name,
		Type:   ty,
		Public: public,
	}
}

func (p *parser) parseImpl() ast.Item {
	spanStart := p.span()

	p.expect("impl")
	generics := p.parseGenerics()
	ty := p.parseType()

	if p.tryConsume("for") {
		trait, ok := ty.(*ast.Path)
		if !ok {
			common.PanicDiag("invalid trait", ty.Span())
		}

		st := p.parseType()

		p.expect(";")

		span := SpanFrom(spanStart, p.prevSpan())
		return ast.NewImplTraitForClass(generics, *trait, st, span)
	}

	p.expect("{")

	var methods []ast.Function

	for !p.Token.Is("}") {
		var attributes []ast.Attribute
		for p.Token.Is("#") {
			attributes = append(attributes, p.parseAttribute())
		}
		method := p.parseClassMethod(false)
		method.Attributes = attributes
		methods = append(methods, method)
	}

	p.expect("}")

	span := SpanFrom(spanStart, p.prevSpan())

	return ast.NewImplClass(generics, ty, methods, span)
}

func (p *parser) parseClassMethod(bodyOptional bool) ast.Function {
	spanStart := p.span()

	p.expect("func")
	name := p.expectIdent()

	sig := p.parseFunctionSignature(
		FlagFuncParamVarArg |
			FlagFuncParamSelf |
			FlagFuncParamNamed,
	)

	var body *ast.Block
	if !bodyOptional {
		b := p.parseBlock()
		body = &b
	} else {
		if p.Token.Is("{") {
			b := p.parseBlock()
			body = &b
		} else {
			p.expect(";")
		}
	}

	span := SpanFrom(spanStart, p.prevSpan())

	return *ast.NewFunction(&name, sig, body, nil, span)
}

func (p *parser) parseTrait() ast.Item {
	spanStart := p.span()

	p.expect("trait")

	name := p.expectIdentMsg("expected trait name")

	var superTraits []ast.Path
	if p.tryConsume(":") {
		for {
			superTrait := p.parsePath()
			superTraits = append(superTraits, superTrait)
			if !p.tryConsume("+") {
				break
			}
		}
	}

	p.expect("{")

	var methods []ast.Function

	for !p.Token.Is("}") {
		method := p.parseClassMethod(true)
		methods = append(methods, method)
	}

	p.expect("}")

	span := SpanFrom(spanStart, p.prevSpan())

	return ast.NewTrait(name, superTraits, methods, span)
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
