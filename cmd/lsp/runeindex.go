package lsp

import (
	"github.com/gluax-lang/gluax/frontend/lexer"
	"github.com/gluax-lang/lsp"
)

type RuneInfo struct {
	Rune     rune
	Position lsp.Position
}

type RuneIndex struct {
	Runes     []RuneInfo
	LineRunes map[uint32][]int
}

func BuildRuneIndex(text string) RuneIndex {
	var runes []RuneInfo
	lineRunes := make(map[uint32][]int)

	lx := lexer.NewLexer("", text)

	for {
		lx.SkipWs()
		c := lx.CurChr
		if c == nil {
			break
		}

		runeIndex := len(runes)
		runes = append(runes, RuneInfo{
			Rune:     *c,
			Position: lsp.Position{Line: lx.Line, Character: lx.ColumnUTF16},
		})

		lineRunes[lx.Line] = append(lineRunes[lx.Line], runeIndex)

		lx.Advance()
	}

	return RuneIndex{
		Runes:     runes,
		LineRunes: lineRunes,
	}
}

func (ri *RuneIndex) GetRuneAt(pos lsp.Position) *RuneInfo {
	runeIndices, exists := ri.LineRunes[pos.Line]
	if !exists {
		return nil
	}

	for _, idx := range runeIndices {
		r := &ri.Runes[idx]
		if r.Position.Character == pos.Character {
			return r
		}
	}
	return nil
}

func (ri *RuneIndex) GetRuneBefore(pos lsp.Position) *RuneInfo {
	// Check current line first
	if runeIndices, exists := ri.LineRunes[pos.Line]; exists {
		for i := len(runeIndices) - 1; i >= 0; i-- {
			r := &ri.Runes[runeIndices[i]]
			if r.Position.Character < pos.Character {
				return r
			}
		}
	}

	// If no rune found on current line, check previous lines
	for line := pos.Line - 1; line >= 0; line-- {
		if runeIndices, exists := ri.LineRunes[line]; exists && len(runeIndices) > 0 {
			// Return the last rune on this line
			lastIdx := runeIndices[len(runeIndices)-1]
			return &ri.Runes[lastIdx]
		}
	}

	return nil
}
