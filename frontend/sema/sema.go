package sema

import (
	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/common"
	protocol "github.com/gluax-lang/lsp"
)

type Span = common.Span
type Diagnostic = protocol.Diagnostic
type InlayHint = protocol.InlayHint

type Symbol = ast.Symbol
type Value = ast.Value
type Type = ast.SemType

const SymValue = ast.SymValue
const SymType = ast.SymType
const SymImport = ast.SymImport
const SymTrait = ast.SymTrait

type ImportInfo = ast.SemImport

type SemStruct = ast.SemStruct
