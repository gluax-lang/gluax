package preprocess

import (
	"bufio"
	"regexp"
	"strings"

	"maps"

	"github.com/gluax-lang/gluax/frontend/common"
	protocol "github.com/gluax-lang/lsp"
)

type Span = common.Span
type diagnostic = protocol.Diagnostic

func Preprocess(input string, defaultMacros map[string]string) (string, *diagnostic) {
	defineRe := regexp.MustCompile(`^#define\s+(\w+)(?:\s+(.*))?$`)
	ifdefRe := regexp.MustCompile(`^#ifdef\s+(\w+)$`)
	elseRe := regexp.MustCompile(`^#else$`)
	endifRe := regexp.MustCompile(`^#endif$`)
	macroRe := regexp.MustCompile(`\b(\w+)\b`)
	stringRe := regexp.MustCompile(`"[^"]*"`)

	macros := make(map[string]string, len(defaultMacros))
	maps.Copy(macros, defaultMacros)

	type condState struct {
		active bool
		span   Span
	}
	var condStack []condState
	var outputLines []string

	throwErr := func(msg string, lineNum uint32, line string) (string, *diagnostic) {
		span := common.SpanNew(lineNum, lineNum, 0, uint32(len(line)))
		return "", common.ErrorDiag(msg, span)
	}

	scanner := bufio.NewScanner(strings.NewReader(input))
	lineNum := uint32(0)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimLeft(line, " \t")

		// Determine which, if any, directive this is
		isDefine := defineRe.MatchString(trimmed)
		isIfdef := ifdefRe.MatchString(trimmed)
		isElse := elseRe.MatchString(trimmed)
		isEndif := endifRe.MatchString(trimmed)

		if strings.HasPrefix(trimmed, "#") && (isDefine || isIfdef || isElse || isEndif) {
			switch {
			case isDefine:
				// #define
				allActive := true
				for _, cs := range condStack {
					if !cs.active {
						allActive = false
						break
					}
				}
				if allActive {
					caps := defineRe.FindStringSubmatch(trimmed)
					name := caps[1]
					value := ""
					if len(caps) > 2 {
						value = strings.TrimSpace(caps[2])
					}
					macros[name] = value
				}
				outputLines = append(outputLines, "")

			case isIfdef:
				// #ifdef
				caps := ifdefRe.FindStringSubmatch(trimmed)
				name := caps[1]
				parentActive := true
				for _, cs := range condStack {
					if !cs.active {
						parentActive = false
						break
					}
				}
				_, macroExists := macros[name]
				isActive := parentActive && macroExists
				span := common.SpanNew(lineNum, lineNum, 0, uint32(len(line)))
				condStack = append(condStack, condState{isActive, span})
				outputLines = append(outputLines, "")

			case isElse:
				// #else
				if len(condStack) == 0 {
					return throwErr("#else without matching #ifdef", lineNum, line)
				}
				parentActive := true
				if len(condStack) > 1 {
					for _, cs := range condStack[:len(condStack)-1] {
						if !cs.active {
							parentActive = false
							break
						}
					}
				}
				top := &condStack[len(condStack)-1]
				top.active = parentActive && !top.active
				outputLines = append(outputLines, "")

			case isEndif:
				// #endif
				if len(condStack) == 0 {
					return throwErr("#endif without matching #ifdef", lineNum, line)
				}
				condStack = condStack[:len(condStack)-1]
				outputLines = append(outputLines, "")
			}
		} else {
			// Non-directive or unknown directive: process normally
			allActive := true
			for _, cs := range condStack {
				if !cs.active {
					allActive = false
					break
				}
			}
			if allActive {
				var result strings.Builder
				lastEnd := 0
				for _, loc := range stringRe.FindAllStringIndex(line, -1) {
					segment := line[lastEnd:loc[0]]
					replaced := macroRe.ReplaceAllStringFunc(segment, func(word string) string {
						if val, ok := macros[word]; ok && val != "" {
							return val
						}
						return word
					})
					result.WriteString(replaced)
					result.WriteString(line[loc[0]:loc[1]])
					lastEnd = loc[1]
				}
				if lastEnd < len(line) {
					segment := line[lastEnd:]
					replaced := macroRe.ReplaceAllStringFunc(segment, func(word string) string {
						if val, ok := macros[word]; ok && val != "" {
							return val
						}
						return word
					})
					result.WriteString(replaced)
				}
				outputLines = append(outputLines, result.String())
			} else {
				// Inactive block
				outputLines = append(outputLines, "")
			}
		}
	}

	if len(condStack) > 0 {
		return "", common.ErrorDiag("Unclosed #ifdef block", common.SpanDefault())
	}
	return strings.Join(outputLines, "\n"), nil
}
