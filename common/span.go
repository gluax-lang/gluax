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
	ID                     uint64
	LineStart, LineEnd     uint32
	ColumnStart, ColumnEnd uint32
	Source                 string // nil == unknown
}

func (s Span) ToRange() protocol.Range {
	return protocol.Range{
		Start: protocol.Position{
			Line:      s.LineStart,
			Character: s.ColumnStart,
		},
		End: protocol.Position{
			Line:      s.LineEnd,
			Character: s.ColumnEnd,
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

// NewDefault Default span (1:1).
func SpanDefault() Span {
	return Span{
		ID:          nextSpanID(),
		LineStart:   0,
		LineEnd:     0,
		ColumnStart: 0,
		ColumnEnd:   0,
	}
}

func SpanNew(lineStart, lineEnd, columnStart, columnEnd uint32) Span {
	return Span{
		ID:          nextSpanID(),
		LineStart:   lineStart,
		LineEnd:     lineEnd,
		ColumnStart: columnStart,
		ColumnEnd:   columnEnd,
	}
}

func SpanSrc(src string) Span {
	return Span{
		ID:          nextSpanID(),
		LineStart:   0,
		LineEnd:     0,
		ColumnStart: 0,
		ColumnEnd:   0,
		Source:      src,
	}
}

// From joins the outer bounds of two spans.
func SpanFrom(start, end Span) Span {
	return Span{
		ID:          nextSpanID(),
		LineStart:   start.LineStart,
		LineEnd:     end.LineEnd,
		ColumnStart: start.ColumnStart,
		ColumnEnd:   end.ColumnEnd,
		Source:      start.Source,
	}
}
