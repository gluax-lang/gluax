package sema

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"

	"slices"

	"github.com/gluax-lang/gluax/frontend"
	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/common"
	"github.com/gluax-lang/gluax/frontend/lexer"
	"github.com/gluax-lang/gluax/frontend/parser"
	"github.com/gluax-lang/gluax/frontend/preprocess"
	"github.com/gluax-lang/gluax/std"
	protocol "github.com/gluax-lang/lsp"
)

type State struct {
	Label     string               // "SERVER" or "CLIENT"
	Macros    map[string]string    // e.g. {"SERVER": ""}, {"CLIENT": ""}
	RootScope *Scope               // which root scope we attach to in this pass
	Files     map[string]*Analysis // where we store the resulting analyses
}

func NewState(label string, scope *Scope) *State {
	return &State{
		Label:     label,
		Macros:    make(map[string]string),
		RootScope: scope,
		Files:     make(map[string]*Analysis),
	}
}

func createImport(name string, analysis *Analysis) ast.SemImport {
	tokStr := lexer.NewTokString(name, common.SpanDefault())
	tokIdent := lexer.NewTokIdent(name, common.SpanDefault())
	importDef := ast.NewImport(tokStr, &tokIdent, common.SpanDefault())
	return ast.NewSemImport(*importDef, name, analysis)
}

var (
	stdInitOnce        sync.Once
	stdProjectAnalysis *ProjectAnalysis
	stdServerScope     *Scope
	stdClientScope     *Scope
)

func initStd() {
	stdInitOnce.Do(func() {
		var err error
		stdProjectAnalysis, err = AnalyzeProject(std.Workspace, std.Files)
		if err != nil {
			panic(fmt.Sprintf("failed to analyze std project: %v", err))
		}
		var buildStdScope = func(files map[string]*Analysis) *Scope {
			mainAnalysis := files[stdProjectAnalysis.Main]
			stdImport := createImport("std", mainAnalysis)
			scope := NewScope(nil)
			for name, sym := range mainAnalysis.Scope.Parent.Symbols {
				if ast.IsBuiltinType(sym.Name) {
					scope.Symbols[name] = sym
				}
			}
			publicPath := stdProjectAnalysis.StripWorkspace(filepath.Join("src", "public.gluax"))
			publicAnalysis := files[publicPath]
			for name, sym := range publicAnalysis.Scope.Symbols {
				if sym.IsPublic() {
					scope.Symbols[name] = sym
				}
			}
			_ = scope.AddImport("std", stdImport, common.SpanDefault(), true)
			return scope
		}
		stdServerScope = buildStdScope(stdProjectAnalysis.ServerFiles())
		stdClientScope = buildStdScope(stdProjectAnalysis.ClientFiles())
	})
}

func GetStdProjectAnalysis() *ProjectAnalysis {
	initStd()
	return stdProjectAnalysis
}

func GetStdServerScope() *Scope {
	initStd()
	return stdServerScope
}

func GetStdClientScope() *Scope {
	initStd()
	return stdClientScope
}

// ProjectAnalysis manages analysis of an entire workspace.
type ProjectAnalysis struct {
	Main      string // path to main.gluax
	Config    frontend.GluaxToml
	workspace string
	overrides map[string]string

	// Name of the current package being analyzed
	currentPackage string

	// Processing Globals declarations or not?
	processingGlobals bool

	// Either "SERVER" or "CLIENT"; used inside AnalyzeFile
	serverState  *State
	clientState  *State
	currentState *State

	// After merging, final map that combines them
	files map[string]*Analysis
}

// NewProjectAnalysis builds a project-level container.
func NewProjectAnalysis(workspace string, overrides map[string]string) *ProjectAnalysis {
	pa := &ProjectAnalysis{
		workspace: workspace,
		overrides: make(map[string]string),

		// Final merged map
		files: make(map[string]*Analysis),
	}

	for p, c := range overrides {
		if p != "" {
			pa.overrides[common.FilePathClean(p)] = c
		}
	}

	return pa
}

func (pa *ProjectAnalysis) getStateFiles() map[string]*Analysis {
	return pa.currentState.Files
}

func (pa *ProjectAnalysis) globalsList() []string {
	out := make([]string, 0, 60)
	if pa.Config.Std && pa.workspace == std.Workspace { // std build: collect from std/globals/*.gluax in overrides
		const (
			prefix = "std/globals/"
			suffix = ".gluax"
		)
		for p := range pa.overrides {
			if strings.HasPrefix(p, prefix) && strings.HasSuffix(p, suffix) {
				out = append(out, p)
			}
		}
	} else {
		globalsDir := filepath.Join(pa.workspace, "globals")
		entries, err := os.ReadDir(globalsDir)
		if err != nil {
			return out
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if strings.HasSuffix(e.Name(), ".gluax") {
				name := common.FilePathClean(filepath.Join(globalsDir, e.Name()))
				out = append(out, name) // full absolute path
			}
		}
	}
	return out
}

func (pa *ProjectAnalysis) newAnalysis(path string) *Analysis {
	return &Analysis{
		Workspace: pa.workspace,
		Src:       path,
		Scope:     NewScope(pa.currentState.RootScope),
		Project:   pa,
	}
}

func (pa *ProjectAnalysis) AnalyzeFile(path string) (analysis *Analysis, hardError bool) {
	path = common.FilePathClean(path)
	m := pa.getStateFiles()

	if existing, ok := m[path]; ok {
		// Already analyzed under this pass (SERVER or CLIENT)
		return existing, false
	}

	// Otherwise, create a new Analysis
	analysis = pa.newAnalysis(path)
	m[path] = analysis // store it so we don't re-analyze under same state

	// Load file content (override or from disk)
	code, err := pa.loadFileContent(path)
	if err != nil {
		hardError = true
		analysis.Error(fmt.Sprintf("Failed to load file: %v", err), common.SpanDefault())
		return
	}

	macros := map[string]string{
		pa.currentState.Label: "",
	}
	preprocessed, diag := preprocess.Preprocess(code, macros)
	if diag != nil {
		hardError = true
		analysis.Diags = append(analysis.Diags, *diag)
		return
	}

	toks, diag := lexer.Lex(path, preprocessed)
	if diag != nil {
		hardError = true
		analysis.Diags = append(analysis.Diags, *diag)
		return
	}

	astRoot, diag := parser.Parse(toks, pa.processingGlobals)
	if diag != nil {
		hardError = true
		analysis.Diags = append(analysis.Diags, *diag)
		return
	}
	analysis.Ast = astRoot

	defer func() {
		if r := recover(); r != nil {
			hardError = true
			if errStr, ok := r.(string); ok {
				if errStr != "" {
					log.Printf("panic: %v\n%s", errStr, debug.Stack())
					analysis.Error(errStr, common.SpanDefault())
				}
			} else {
				log.Printf("panic: %v\n%s", r, debug.Stack())
				analysis.Error(fmt.Sprintf("%v", r), common.SpanDefault())
			}
		}
	}()

	analysis.handleAst(astRoot)
	return
}

// loadFileContent checks if path is overridePath; else read from disk
func (pa *ProjectAnalysis) loadFileContent(path string) (string, error) {
	path = common.FilePathClean(path)
	if content, ok := pa.overrides[path]; ok {
		return content, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (pa *ProjectAnalysis) Files() map[string]*Analysis {
	return pa.files
}

func (pa ProjectAnalysis) ServerFiles() map[string]*Analysis {
	return pa.serverState.Files
}

func (pa *ProjectAnalysis) ClientFiles() map[string]*Analysis {
	return pa.clientState.Files
}

func (pa *ProjectAnalysis) importGlobals() {
	pa.processingGlobals = true
	globalsA := pa.newAnalysis("globals")
	for _, path := range pa.globalsList() {
		analysis, _ := pa.AnalyzeFile(path)
		inferredName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if !lexer.IsValidIdent(inferredName) {
			analysis.Error(fmt.Sprintf("`%s` is not a valid identifier to use, rename it", inferredName), common.SpanDefault())
			continue
		}
		createdImport := createImport(inferredName, analysis)
		_ = globalsA.Scope.AddImport(inferredName, createdImport, common.SpanDefault(), true)
	}
	globalsImport := createImport("globals", globalsA)
	err := pa.currentState.RootScope.AddImport("globals", globalsImport, common.SpanDefault(), true)
	if err != nil {
		panic(err)
	}
	pa.processingGlobals = false
}

func (pa *ProjectAnalysis) processState(state *State, mainPath string) {
	pa.currentState = state

	if pa.Config.Std {
		state.RootScope = NewScope(nil)
	} else {
		switch state {
		case pa.serverState:
			state.RootScope = NewScope(GetStdServerScope())
		case pa.clientState:
			state.RootScope = NewScope(GetStdClientScope())
		default:
			panic(fmt.Sprintf("unknown state: %s", state.Label))
		}
	}

	if pa.Config.Std {
		mainPath = common.FilePathClean(mainPath)
	}

	// process globals, if doing std, then std will manually call it
	if !pa.Config.Std {
		pa.importGlobals()
	}

	_, _ = pa.AnalyzeFile(mainPath)

	// don't leak full path
	for p, a := range state.Files {
		delete(state.Files, p)
		state.Files[pa.StripWorkspace(p)] = a
	}
}

func AnalyzeProject(workspace string, overrides map[string]string) (*ProjectAnalysis, error) {
	pa := NewProjectAnalysis(workspace, overrides)

	mainPath := filepath.Join(workspace, "src", "main.gluax")

	{
		tomlContent, err := pa.loadFileContent(filepath.Join(workspace, "gluax.toml"))
		if err != nil {
			return nil, fmt.Errorf("failed to load gluax.toml: %w", err)
		}
		gluaxToml, err := frontend.HandleGluaxToml(tomlContent)
		if err != nil {
			return nil, fmt.Errorf("failed to load gluax.toml: %w", err)
		} else {
			if gluaxToml.Lib {
				mainPath = filepath.Join(workspace, "src", "lib.gluax")
			}
		}
		pa.Config = gluaxToml
	}

	if !pa.Config.Std {
		GetStdProjectAnalysis()
	}

	pa.serverState = NewState("SERVER", nil)
	pa.clientState = NewState("CLIENT", nil)

	mainPath = common.FilePathClean(mainPath)

	pa.currentPackage = pa.Config.Name
	pa.Main = pa.StripWorkspace(mainPath)

	pa.processState(pa.serverState, mainPath)
	pa.processState(pa.clientState, mainPath)

	// Now unify pa.filesServer and pa.filesClient into pa.files
	pa.mergeAll()

	for _, path := range pa.globalsList() {
		path = pa.StripWorkspace(path)
		delete(pa.serverState.Files, path)
		delete(pa.clientState.Files, path)
	}

	return pa, nil
}

func (pa *ProjectAnalysis) mergeAll() {
	serverFiles := pa.serverState.Files
	clientFiles := pa.clientState.Files
	paths := make(map[string]struct{}, len(serverFiles)+len(clientFiles))
	for p := range serverFiles {
		paths[p] = struct{}{}
	}
	for p := range clientFiles {
		paths[p] = struct{}{}
	}
	for p := range paths {
		srv, cli := serverFiles[p], clientFiles[p]
		switch {
		case srv != nil && cli != nil:
			pa.files[p] = mergeAnalysisResults(srv, cli)
		case srv != nil: // server‑only
			pa.files[p] = annotateSingleState(srv, "(SERVER) ", ":🔹")
		case cli != nil: // client‑only
			pa.files[p] = annotateSingleState(cli, "(CLIENT) ", ":🔸")
		}
	}
}

func addPrefix(pfx string, diags []protocol.Diagnostic) []protocol.Diagnostic {
	out := make([]protocol.Diagnostic, len(diags))
	for i, d := range diags {
		c := d
		c.Message = pfx + strings.TrimSpace(c.Message)
		out[i] = c
	}
	return out
}

func annotateSingleState(src *Analysis, pfx, glyph string) *Analysis {
	out := *src
	out.Diags = addPrefix(pfx, src.Diags)
	out.InlayHints = slices.Clone(src.InlayHints)
	for i := range out.InlayHints {
		if len(out.InlayHints[i].Label) > 0 {
			lp := &out.InlayHints[i].Label[0]
			lp.Value = glyph + strings.TrimSpace(lp.Value)
		}
	}
	return &out
}

func mergeAnalysisResults(srvA, cliA *Analysis) *Analysis {
	merged := &Analysis{}

	// Diagnostics
	merged.Diags = append(merged.Diags, addPrefix("(SERVER) ", srvA.Diags)...)
	merged.Diags = append(merged.Diags, addPrefix("(CLIENT) ", cliA.Diags)...)

	// Inlay hints
	type key struct{ line, char uint32 }
	type pair struct{ srv, cli string }

	pairs := map[key]*pair{}
	collect := func(hints []protocol.InlayHint, isSrv bool) {
		for _, h := range hints {
			k := key{h.Position.Line, h.Position.Character}
			lbl := ""
			for _, p := range h.Label {
				lbl += p.Value
			}
			if pairs[k] == nil {
				pairs[k] = &pair{}
			}
			if isSrv {
				pairs[k].srv = strings.TrimSpace(lbl)
			} else {
				pairs[k].cli = strings.TrimSpace(lbl)
			}
		}
	}
	collect(srvA.InlayHints, true)
	collect(cliA.InlayHints, false)

	const (
		sharedGlyph = ": "
		serverGlyph = ":🔹"
		clientGlyph = ":🔸"
	)

	for k, p := range pairs {
		var label string
		switch {
		case p.srv == "" && p.cli != "":
			label = clientGlyph + p.cli
		case p.cli == "" && p.srv != "":
			label = serverGlyph + p.srv
		case p.srv == p.cli:
			label = sharedGlyph + p.srv
		default:
			label = ":🔹" + p.srv + " • " + p.cli + "🔸"
		}

		merged.InlayHints = append(merged.InlayHints, protocol.InlayHint{
			Position: protocol.Position{Line: k.line, Character: k.char},
			Label:    []protocol.InlayHintLabelPart{{Value: label}},
		})
	}

	return merged
}
