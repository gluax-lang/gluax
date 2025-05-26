package preprocess

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"

	"maps"

	"github.com/gluax-lang/gluax/frontend/common"
	protocol "github.com/gluax-lang/lsp"
)

type Span = common.Span
type diagnostic = protocol.Diagnostic

var (
	definePattern = regexp.MustCompile(`^#define\s+(\w+)(?:\s+(.*))?$`)
	ifdefPattern  = regexp.MustCompile(`^#ifdef\s+(\w+)$`)
	ifndefPattern = regexp.MustCompile(`^#ifndef\s+(\w+)$`)
	elsePattern   = regexp.MustCompile(`^#else$`)
	elifPattern   = regexp.MustCompile(`^#elif\s+(\w+)$`)
	endifPattern  = regexp.MustCompile(`^#endif$`)
	undefPattern  = regexp.MustCompile(`^#undef\s+(\w+)$`)
	macroPattern  = regexp.MustCompile(`\b(\w+)\b`)
	stringPattern = regexp.MustCompile(`"[^"]*"`)
)

var disallowedMacros = map[string]struct{}{
	"__LINE__": {},
}

func isDisallowedMacro(name string) bool {
	_, disallowed := disallowedMacros[name]
	return disallowed
}

// Preprocess processes input text with C-style preprocessor directives
func Preprocess(input string, defaultMacros map[string]string) (string, *diagnostic) {
	macros := make(map[string]string, len(defaultMacros))
	maps.Copy(macros, defaultMacros)

	processor := &preprocessor{
		macros:      macros,
		condStack:   make([]condState, 0),
		outputLines: make([]string, 0),
	}

	return processor.process(input)
}

type condState struct {
	active      bool
	hasBeenTrue bool // Track if any branch has been active
	span        Span
}

type preprocessor struct {
	macros         map[string]string
	condStack      []condState
	outputLines    []string
	currentLineNum uint32
}

func (p *preprocessor) process(input string) (string, *diagnostic) {
	scanner := bufio.NewScanner(strings.NewReader(input))
	lineNum := uint32(0)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimLeft(line, " \t")

		if err := p.processLine(line, trimmed, lineNum); err != nil {
			return "", err
		}
	}

	if len(p.condStack) > 0 {
		return "", common.ErrorDiag("Unclosed #ifdef block", common.SpanDefault())
	}

	return strings.Join(p.outputLines, "\n"), nil
}

func (p *preprocessor) processLine(line, trimmed string, lineNum uint32) *diagnostic {
	p.currentLineNum = lineNum

	// Check if this is a preprocessor directive
	if strings.HasPrefix(trimmed, "#") {
		return p.processDirective(line, trimmed, lineNum)
	}

	// Process regular line with macro substitution
	p.processRegularLine(line)
	return nil
}

func (p *preprocessor) processDirective(line, trimmed string, lineNum uint32) *diagnostic {
	switch {
	case definePattern.MatchString(trimmed):
		return p.handleDefine(trimmed, lineNum, line)
	case undefPattern.MatchString(trimmed):
		return p.handleUndef(trimmed)
	case ifdefPattern.MatchString(trimmed):
		return p.handleIfdef(trimmed, lineNum, line)
	case ifndefPattern.MatchString(trimmed):
		return p.handleIfndef(trimmed, lineNum, line)
	case elifPattern.MatchString(trimmed):
		return p.handleElif(trimmed, lineNum, line)
	case elsePattern.MatchString(trimmed):
		return p.handleElse(lineNum, line)
	case endifPattern.MatchString(trimmed):
		return p.handleEndif(lineNum, line)
	default:
		// Unknown directive, process as regular line
		p.processRegularLine(line)
		return nil
	}
}

func (p *preprocessor) handleDefine(trimmed string, lineNum uint32, line string) *diagnostic {
	if p.isAllActive() {
		caps := definePattern.FindStringSubmatch(trimmed)
		name := caps[1]
		if p.isMacroDefined(name) {
			return p.throwErr("Macro '"+name+"' is already defined", lineNum, line)
		}
		value := ""
		if len(caps) > 2 {
			value = strings.TrimSpace(caps[2])
		}
		p.macros[name] = value
	}
	p.outputLines = append(p.outputLines, "")
	return nil
}

func (p *preprocessor) handleUndef(trimmed string) *diagnostic {
	if p.isAllActive() {
		caps := undefPattern.FindStringSubmatch(trimmed)
		name := caps[1]
		if isDisallowedMacro(name) {
			return p.throwErr("Cannot undefine predefined macro '"+name+"'", p.currentLineNum, trimmed)
		}
		delete(p.macros, name)
	}
	p.outputLines = append(p.outputLines, "")
	return nil
}

func (p *preprocessor) handleIfdef(trimmed string, lineNum uint32, line string) *diagnostic {
	caps := ifdefPattern.FindStringSubmatch(trimmed)
	name := caps[1]
	parentActive := p.isAllActive()
	isActive := parentActive && p.isMacroDefined(name)
	span := common.SpanNew(lineNum, lineNum, 0, uint32(len(line)))
	p.condStack = append(p.condStack, condState{isActive, isActive, span})
	p.outputLines = append(p.outputLines, "")
	return nil
}

func (p *preprocessor) handleIfndef(trimmed string, lineNum uint32, line string) *diagnostic {
	caps := ifndefPattern.FindStringSubmatch(trimmed)
	name := caps[1]
	parentActive := p.isAllActive()
	isActive := parentActive && !p.isMacroDefined(name)
	span := common.SpanNew(lineNum, lineNum, 0, uint32(len(line)))
	p.condStack = append(p.condStack, condState{isActive, isActive, span})
	p.outputLines = append(p.outputLines, "")
	return nil
}

func (p *preprocessor) handleElif(trimmed string, lineNum uint32, line string) *diagnostic {
	if len(p.condStack) == 0 {
		return p.throwErr("#elif without matching #ifdef", lineNum, line)
	}

	caps := elifPattern.FindStringSubmatch(trimmed)
	name := caps[1]
	parentActive := p.isParentActive()
	top := &p.condStack[len(p.condStack)-1]

	// Only activate if parent is active, no previous branch was true, and macro exists
	top.active = parentActive && !top.hasBeenTrue && p.isMacroDefined(name)
	if top.active {
		top.hasBeenTrue = true
	}
	p.outputLines = append(p.outputLines, "")
	return nil
}

func (p *preprocessor) handleElse(lineNum uint32, line string) *diagnostic {
	if len(p.condStack) == 0 {
		return p.throwErr("#else without matching #ifdef", lineNum, line)
	}

	parentActive := p.isParentActive()
	top := &p.condStack[len(p.condStack)-1]
	top.active = parentActive && !top.hasBeenTrue
	p.outputLines = append(p.outputLines, "")
	return nil
}

func (p *preprocessor) handleEndif(lineNum uint32, line string) *diagnostic {
	if len(p.condStack) == 0 {
		return p.throwErr("#endif without matching #ifdef", lineNum, line)
	}

	p.condStack = p.condStack[:len(p.condStack)-1]
	p.outputLines = append(p.outputLines, "")
	return nil
}

func (p *preprocessor) processRegularLine(line string) {
	if p.isAllActive() {
		result := p.substituteMacros(line)
		p.outputLines = append(p.outputLines, result)
	} else {
		// Inactive block
		p.outputLines = append(p.outputLines, "")
	}
}

func (p *preprocessor) substituteMacros(line string) string {
	macroReplacer := func(word string) string {
		if word == "__LINE__" {
			return fmt.Sprintf("%d", p.currentLineNum)
		}
		if val, ok := p.macros[word]; ok && val != "" {
			return val
		}
		return word
	}

	var result strings.Builder
	lastEnd := 0

	// Find all string literals to avoid substituting macros inside them
	for _, loc := range stringPattern.FindAllStringIndex(line, -1) {
		// Process segment before string literal
		segment := line[lastEnd:loc[0]]
		replaced := macroPattern.ReplaceAllStringFunc(segment, macroReplacer)
		result.WriteString(replaced)

		// Add string literal as-is
		result.WriteString(line[loc[0]:loc[1]])
		lastEnd = loc[1]
	}

	// Process remaining segment after last string literal
	if lastEnd < len(line) {
		segment := line[lastEnd:]
		replaced := macroPattern.ReplaceAllStringFunc(segment, macroReplacer)
		result.WriteString(replaced)
	}

	return result.String()
}

func (p *preprocessor) isAllActive() bool {
	for _, cs := range p.condStack {
		if !cs.active {
			return false
		}
	}
	return true
}

func (p *preprocessor) isMacroDefined(name string) bool {
	if isDisallowedMacro(name) {
		return true
	}
	_, exists := p.macros[name]
	return exists
}

func (p *preprocessor) isParentActive() bool {
	if len(p.condStack) <= 1 {
		return true
	}

	for _, cs := range p.condStack[:len(p.condStack)-1] {
		if !cs.active {
			return false
		}
	}
	return true
}

func (p *preprocessor) throwErr(msg string, lineNum uint32, line string) *diagnostic {
	span := common.SpanNew(lineNum, lineNum, 0, uint32(len(line)))
	return common.ErrorDiag(msg, span)
}
