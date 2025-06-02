package codegen

import (
	"regexp"
	"strings"

	"github.com/gluax-lang/gluax/frontend/sema"
)

var redundantNewlinesRegex = regexp.MustCompile(`(\r?\n){3,}`)

func removeRedundantBlankLines(s string) string {
	return redundantNewlinesRegex.ReplaceAllString(s, "$1$1")
}

func GenerateProject(pA *sema.ProjectAnalysis) (string, string) {
	serverCg := Codegen{
		bufCtx: bufCtx{
			buf: strings.Builder{},
		},
		publicIndex:      1,
		publicMap:        make(map[string]int),
		generatedStructs: make(map[string]struct{}),
	}
	serverCg.bufCtx.buf.Grow(1024 * 2)
	headers(&serverCg)
	serverCg.handleFiles(pA.ServerFiles())
	serverCg.ln("%s(%s);", RUN_IMPORT, pathToLuaString(pA.Main))
	serverCode := serverCg.bufCtx.buf.String()

	clientCg := Codegen{
		bufCtx: bufCtx{
			buf: strings.Builder{},
		},
		publicIndex:      1,
		publicMap:        make(map[string]int),
		generatedStructs: make(map[string]struct{}),
	}
	clientCg.bufCtx.buf.Grow(1024 * 2)
	headers(&clientCg)
	clientCg.handleFiles(pA.ClientFiles())
	clientCg.ln("%s(%s);", RUN_IMPORT, pathToLuaString(pA.Main))
	clientCode := clientCg.bufCtx.buf.String()

	return removeRedundantBlankLines(serverCode), removeRedundantBlankLines(clientCode)
}

func (cg *Codegen) handleFiles(files map[string]*sema.Analysis) {
	for path, analysis := range files {
		addImport(cg, path, analysis)
	}
}

func headers(cg *Codegen) {
	fastLocalsHeaders(cg)
	cg.writeByte('\n')

	structHeaders(cg)
	cg.writeByte('\n')

	importsHeaders(cg)
	cg.writeByte('\n')

	publicHeaders(cg)
	cg.writeByte('\n')
}

func importsHeaders(cg *Codegen) {
	cg.ln("-- imports")
	cg.ln("local %s = {};", IMPORTS_TBL)
	cg.ln("local %s = function(f)", RUN_IMPORT)
	cg.pushIndent()
	cg.ln("if %s[f] then", IMPORTS_TBL)
	cg.pushIndent()
	cg.ln("local load = %s[f]", IMPORTS_TBL)
	cg.ln("%s[f] = nil", IMPORTS_TBL)
	cg.ln("load()")
	cg.popIndent()
	cg.ln("end")
	cg.popIndent()
	cg.ln("end;")
}

func addImport(cg *Codegen, path string, analysis *sema.Analysis) {
	path = pathToLuaString(path)
	cg.writef("%s[%s] = ", IMPORTS_TBL, path)
	cg.writeString("function()\n")
	cg.pushIndent()
	cg.setAnalysis(analysis)
	{
		cg.pushTempScope()
		oldBuf := cg.newBuf()
		cg.generate()
		generated := cg.restoreBuf(oldBuf)
		cg.emitTempLocals()
		cg.writeString(generated)
	}
	cg.popIndent()
	cg.ln("end;\n")
}
