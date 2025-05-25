// Package common provides common diagnostic objects.
package common

import (
	protocol "github.com/gluax-lang/lsp"
)

type (
	dSeverity  = protocol.DiagnosticSeverity
	diagnostic = protocol.Diagnostic
)

func adjustN(n uint32) uint32 {
	if n <= 1 {
		return 0
	}
	return n - 1
}

func NewDiagnostic(severity dSeverity, message string, span Span) *diagnostic {
	return &protocol.Diagnostic{
		Severity: &severity,
		Message:  message,
		Range:    SpanToRange(span),
	}
}

func ErrorDiag(msg string, span Span) *diagnostic {
	return NewDiagnostic(protocol.DiagnosticSeverityError,
		msg, span)
}

func PanicDiag(msg string, span Span) {
	panic(ErrorDiag(msg, span))
}

func WarningDiag(msg string, span Span) *diagnostic {
	return NewDiagnostic(protocol.DiagnosticSeverityWarning,
		msg, span)
}
