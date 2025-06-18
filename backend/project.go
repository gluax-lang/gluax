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
	serverCode := generateCode(pA.ServerState())
	clientCode := generateCode(pA.ClientState())
	return removeRedundantBlankLines(serverCode), removeRedundantBlankLines(clientCode)
}

func generateCode(state *sema.State) string {
	cg := Codegen{
		bufCtx: bufCtx{
			buf: strings.Builder{},
		},
		publicIndex:      1,
		publicMap:        make(map[string]int),
		generatedClasses: make(map[string]struct{}),
	}
	cg.buf().Grow(1024 * 2)
	headers(&cg)
	cg.handleFiles(state.Files)
	if mainFunc := state.MainFunc; mainFunc != nil {
		cg.ln("%s()", cg.decorateFuncName(mainFunc))
	}
	return cg.buf().String()
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
		cg.generateTraits()
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
