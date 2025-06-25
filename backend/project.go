package codegen

import (
	"regexp"
	"sort"
	"strings"

	"github.com/gluax-lang/gluax/frontend/sema"
)

var redundantNewlinesRegex = regexp.MustCompile(`(\r?\n){3,}`)

func removeRedundantBlankLines(s string) string {
	return redundantNewlinesRegex.ReplaceAllString(s, "$1$1")
}

func GenerateProject(pA *sema.ProjectAnalysis) (string, string) {
	serverCode := generateCode(pA, pA.ServerState())
	// clientCode := generateCode(pA, pA.ClientState())
	return removeRedundantBlankLines(serverCode), removeRedundantBlankLines("clientCode")
}

func newCodegen(pA *sema.ProjectAnalysis) *Codegen {
	cg := Codegen{
		ProjectAnalysis: pA,
		bufCtx: bufCtx{
			buf: strings.Builder{},
		},
		publicIndex:      1,
		publicMap:        make(map[string]int),
		generatedClasses: make(map[string]struct{}),
		usedPublics:      make(map[any]struct{}),
	}
	cg.buf().Grow(1024 * 2)
	return &cg
}

func generateCode(pA *sema.ProjectAnalysis, state *sema.State) string {
	cg := newCodegen(pA)
	if pA.Options.Release {
		cg.usedPublics = checkUsed(pA, state)
	}
	headers(cg)
	cg.handleFiles(state.Files)
	if mainFunc := state.MainFunc; mainFunc != nil {
		cg.ln("%s()", cg.decorateFuncName(mainFunc))
	}
	return cg.buf().String()
}

func checkUsed(pA *sema.ProjectAnalysis, state *sema.State) map[any]struct{} {
	cg := newCodegen(pA)
	cg.checkingUsed = true
	main := state.Files[cg.ProjectAnalysis.Main]
	cg.setAnalysis(main)
	cg.decorateFuncName(state.MainFunc)
	return cg.usedPublics
}

func (cg *Codegen) markUsed(v any) bool {
	if !cg.checkingUsed {
		return true // if we are not checking used, then act like it got used
	}
	_, exists := cg.usedPublics[v]
	cg.usedPublics[v] = struct{}{} // mark it as used
	return exists
}

func (cg *Codegen) isMarkedUsed(v any) bool {
	if !cg.ProjectAnalysis.Options.Release {
		return true
	}
	_, exists := cg.usedPublics[v]
	return exists
}

func (cg *Codegen) canGenerate(v any) bool {
	if !cg.ProjectAnalysis.Options.Release || cg.checkingUsed {
		// if we are checking unused, then we should always return true
		return true
	}
	_, exists := cg.usedPublics[v]
	return exists
}

func (cg *Codegen) handleFiles(files map[string]*sema.Analysis) {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	// Process files in sorted order
	cg.pushTempScope()
	oldBuf := cg.newBuf()
	cg.runGenerationPhase(files, paths, func(cg *Codegen) {
		cg.generateClasses()
	})
	cg.runGenerationPhase(files, paths, func(cg *Codegen) {
		cg.generateTraitImpls()
	})
	cg.runGenerationPhase(files, paths, func(cg *Codegen) {
		cg.generateFunctions()
	})
	cg.runGenerationPhase(files, paths, func(cg *Codegen) {
		cg.generateLets()
	})
	generated := cg.restoreBuf(oldBuf)
	cg.emitTempLocals()
	cg.writeString(generated)
}

func (cg *Codegen) runGenerationPhase(files map[string]*sema.Analysis, paths []string, generateFunc func(*Codegen)) {
	for _, path := range paths {
		// cg.ln("-- %s", path)
		analysis := files[path]
		cg.setAnalysis(analysis)
		generateFunc(cg)
		// cg.ln("-- end %s", path)
	}
}

func headers(cg *Codegen) {
	fastLocalsHeaders(cg)
	cg.writeByte('\n')

	classHeaders(cg)
	cg.writeByte('\n')

	publicHeaders(cg)
	cg.writeByte('\n')
}
