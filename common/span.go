package common

import (
	"fmt"
	"sync/atomic"

	protocol "github.com/gluax-lang/lsp"
)

var globalSpanID uint64

func nextSpanID() uint64 {
	return atomic.AddUint64(&globalSpanID, 1)
}

// Span represents a range in a source file.
type Span struct {
	ID                               uint64
	LineStart, LineEnd               uint32
	ColumnStart, ColumnEnd           uint32
	ColumnStartUTF16, ColumnEndUTF16 uint32 // for LSP, which uses UTF-16 code units
	Source                           string // nil == unknown
}

func (s Span) ToRange() protocol.Range {
	return protocol.Range{
		Start: protocol.Position{
			Line:      s.LineStart,
			Character: s.ColumnStartUTF16,
		},
		End: protocol.Position{
			Line:      s.LineEnd,
			Character: s.ColumnEndUTF16,
		},
	}
}

func (s Span) ToLocation() protocol.Location {
	return protocol.Location{
		URI:   FilePathToURI(s.Source),
		Range: s.ToRange(),
	}
}

func (s Span) String() string {
	return fmt.Sprintf("%d:%d-%d:%d (%s)", s.LineStart, s.ColumnStart, s.LineEnd, s.ColumnEnd, s.Source)
}

func spanCreate(
	lineStart, lineEnd, columnStart, columnEnd, columnStartUTF16, columnEndUTF16 uint32,
	src string,
) Span {
	return Span{
		ID:               nextSpanID(),
		LineStart:        lineStart,
		LineEnd:          lineEnd,
		ColumnStart:      columnStart,
		ColumnEnd:        columnEnd,
		ColumnStartUTF16: columnStartUTF16,
		ColumnEndUTF16:   columnEndUTF16,
		Source:           src,
	}
}

func SpanDefault() Span {
	return spanCreate(0, 0, 0, 0, 0, 0, "")
}

func SpanNew(lineStart, lineEnd, columnStart, columnEnd, columnStartUTF16, columnEndUTF16 uint32) Span {
	return spanCreate(lineStart, lineEnd, columnStart, columnEnd, columnStartUTF16, columnEndUTF16, "")
}

func SpanSrc(src string) Span {
	return spanCreate(0, 0, 0, 0, 0, 0, src)
}

// From joins the outer bounds of two spans.
func SpanFrom(start, end Span) Span {
	return spanCreate(
		start.LineStart,
		end.LineEnd,
		start.ColumnStart,
		end.ColumnEnd,
		start.ColumnStartUTF16,
		end.ColumnEndUTF16,
		start.Source,
	)
}
