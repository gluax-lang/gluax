package sema

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"slices"

	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend"
	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/lexer"
	"github.com/gluax-lang/gluax/frontend/parser"
	"github.com/gluax-lang/gluax/frontend/preprocess"
	"github.com/gluax-lang/gluax/std"
	protocol "github.com/gluax-lang/lsp"
)

func createImport(name string, analysis *Analysis) ast.SemImport {
	tokStr := lexer.NewTokString(name, common.SpanDefault())
	tokIdent := lexer.NewTokIdent(name, common.SpanDefault())
	importDef := ast.NewImport(tokStr, &tokIdent, common.SpanDefault())
	return ast.NewSemImport(*importDef, name, analysis)
}

type ProjectAnalysis struct {
	Main      string // path to main.gluax
	Config    frontend.GluaxToml
	workspace string
	overrides map[string]string

	// Processing Globals declarations or not?
	processingGlobals bool

	OsRoot *os.Root

	// Either "SERVER" or "CLIENT"; used inside AnalyzeFile
	serverState  *State
	clientState  *State
	currentState *State

	// After merging, final map that combines them
	files map[string]*Analysis

	allGlobals []string
}

// NewProjectAnalysis builds a project-level container.
func NewProjectAnalysis(workspace string, overrides map[string]string) *ProjectAnalysis {
	pa := &ProjectAnalysis{
		workspace: workspace,
		overrides: make(map[string]string),

		// Final merged map
		files: make(map[string]*Analysis),

		allGlobals: make([]string, 0, 10),
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
	if pa.Config.Std && pa.workspace == std.Workspace {
		const (
			prefix = "std/src/@globals/"
			suffix = ".gluax"
		)
		for p := range pa.overrides {
			if strings.HasPrefix(p, prefix) && strings.HasSuffix(p, suffix) {
				out = append(out, p)
				pa.allGlobals = append(pa.allGlobals, pa.PathRelativeToWorkspace(p)) // keep for later
			}
		}
	} else {
		globalsDir := filepath.Join(pa.workspace, "src", "@globals")
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
				out = append(out, name)                                                 // full absolute path
				pa.allGlobals = append(pa.allGlobals, pa.PathRelativeToWorkspace(name)) // keep for later
			}
		}
	}
	return out
}

func (pa *ProjectAnalysis) newAnalysis(path string) *Analysis {
	scope := pa.currentState.RootScope
	if path != "types" {
		scope = NewScope(pa.currentState.RootScope)
	}
	return &Analysis{
		Workspace: pa.workspace,
		Src:       path,
		Scope:     scope,
		Project:   pa,
		State:     pa.currentState,
	}
}

func (pa *ProjectAnalysis) AnalyzeFile(path string) (analysis *Analysis, hardError bool) {
	path = common.FilePathClean(path)
	m := pa.getStateFiles()

	// println(false, path)
	if existing, ok := m[path]; ok {
		// Already analyzed under this pass (SERVER or CLIENT)
		return existing, false
	}

	// Otherwise, create a new Analysis
	analysis = pa.newAnalysis(path)
	m[path] = analysis // store it so we don't re-analyze under same state

	// Load file content (override or from disk)
	code, err := pa.ReadFile(path)
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

func (pa *ProjectAnalysis) ReadFile(path string) (string, error) {
	path = common.FilePathClean(path)
	if content, ok := pa.overrides[path]; ok {
		return content, nil
	}

	path = pa.StripWorkspace(path)

	f, err := pa.OsRoot.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return string(content), nil
}

func (pa *ProjectAnalysis) Files() map[string]*Analysis {
	return pa.files
}

func (pa ProjectAnalysis) ServerState() *State {
	return pa.serverState
}

func (pa ProjectAnalysis) ServerFiles() map[string]*Analysis {
	return pa.serverState.Files
}

func (pa ProjectAnalysis) ClientState() *State {
	return pa.clientState
}

func (pa *ProjectAnalysis) ClientFiles() map[string]*Analysis {
	return pa.clientState.Files
}

func (pa *ProjectAnalysis) CurrentPackage() string {
	return pa.Config.Name
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
	globalsScope := NewScope(pa.currentState.RootScope)
	err := globalsScope.AddImport("globals", globalsImport, common.SpanDefault(), true)
	if err != nil {
		panic(err)
	}
	pa.currentState.RootScope = globalsScope
	pa.processingGlobals = false
}

func (pa *ProjectAnalysis) getProjectConfig(workspace string) (frontend.GluaxToml, error) {
	config := frontend.GluaxToml{}
	tomlContent, err := pa.ReadFile(filepath.Join(workspace, "gluax.toml"))
	if err != nil {
		return config, fmt.Errorf("failed to load gluax.toml: %w", err)
	}
	gluaxToml, err := frontend.HandleGluaxToml(tomlContent)
	if err != nil {
		return config, fmt.Errorf("failed to load gluax.toml: %w", err)
	}
	return gluaxToml, nil
}

func (pa *ProjectAnalysis) SetRoot(workspace string) (func(), error) {
	root, err := os.OpenRoot(workspace)
	if err != nil {
		return nil, fmt.Errorf("failed to open workspace root: %w", err)
	}
	oldRoot := pa.OsRoot
	pa.OsRoot = root
	return func() {
		root.Close()
		pa.OsRoot = oldRoot
	}, nil
}

func (pa *ProjectAnalysis) processPackage(pkgPath string, realPath bool) error {
	oldWs, oldConfig, oldRootScope := pa.workspace, pa.Config, pa.currentState.RootScope
	pa.workspace = pkgPath

	mainPath := filepath.Join(pkgPath, "src", "main.gluax")

	isMain := pa.workspace == oldWs

	if !isMain {
		if realPath {
			restoreRoot, err := pa.SetRoot(pkgPath)
			if err != nil {
				return fmt.Errorf("failed to set root: %w", err)
			}
			defer restoreRoot()
		}

		var err error
		pa.Config, err = pa.getProjectConfig(pkgPath)
		if err != nil {
			return fmt.Errorf("failed to load project config: %w", err)
		}
	}

	if pa.Config.Lib {
		mainPath = filepath.Join(pkgPath, "src", "lib.gluax")
	}

	mainPath = common.FilePathClean(mainPath)
	pa.Main = pa.PathRelativeToWorkspace(mainPath)

	pa.importGlobals()

	analysis, _ := pa.AnalyzeFile(mainPath)

	state := pa.currentState
	// don't leak full path
	for p, a := range state.Files {
		if !pa.StartsWithWorkspace(p) {
			continue
		}
		delete(state.Files, p)
		state.Files[pa.PathRelativeToWorkspace(p)] = a
	}

	packageName := pa.CurrentPackage()

	pa.workspace, pa.Config, pa.currentState.RootScope = oldWs, oldConfig, oldRootScope

	if !isMain {
		imp := createImport(packageName, analysis)
		err := state.RootScope.AddImport(packageName, imp, common.SpanDefault(), true)
		if err != nil {
			panic(err)
		}
	}

	return nil
}

func (pa *ProjectAnalysis) processState(state *State, workspace string) error {
	pa.currentState = state
	_, _ = pa.AnalyzeFile("types")
	// if we are processing std, then don't process std twice
	if !pa.Config.Std {
		// keeping these comments for future reference
		// stdPath := "full std path"
		// stdPath = common.FilePathClean(stdPath)
		oldOverrides := pa.overrides
		pa.overrides = std.Files
		if err := pa.processPackage(std.Workspace, false); err != nil {
			return err
		}
		pa.overrides = oldOverrides
		publicPath := common.FilePathClean(filepath.Join("std", "src", "public.gluax"))
		publicAnalysis := pa.currentState.Files[publicPath]
		for name, sym := range publicAnalysis.Scope.Symbols {
			if sym.IsPublic() {
				pa.currentState.RootScope.Symbols[name] = sym
			}
		}
	}
	if err := pa.processPackage(workspace, true); err != nil {
		return err
	}
	return nil
}

func AnalyzeProject(workspace string, overrides map[string]string) (*ProjectAnalysis, error) {
	overrides["types"] = ast.BuiltinTypes
	pa := NewProjectAnalysis(workspace, overrides)

	restoreRoot, err := pa.SetRoot(workspace)
	if err != nil {
		return nil, fmt.Errorf("failed to set root: %w", err)
	}
	defer restoreRoot()

	pa.Config, err = pa.getProjectConfig(workspace)
	if err != nil {
		return nil, fmt.Errorf("failed to load project config: %w", err)
	}

	pa.serverState = NewState("SERVER")
	pa.clientState = NewState("CLIENT")

	if err := pa.processState(pa.serverState, workspace); err != nil {
		return nil, err
	}
	if err := pa.processState(pa.clientState, workspace); err != nil {
		return nil, err
	}

	// Now unify pa.filesServer and pa.filesClient into pa.files
	pa.mergeAll()

	for _, path := range pa.allGlobals {
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
	if srvA.Src != "" {
		merged.Src = srvA.Src
	} else {
		merged.Src = cliA.Src
	}

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
