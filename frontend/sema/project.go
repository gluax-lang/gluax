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

const typesFile = "__builtin_types__"

func createImport(name string, analysis *Analysis) ast.SemImport {
	tokStr := lexer.NewTokString(name, common.SpanDefault())
	tokIdent := lexer.NewTokIdent(name, common.SpanDefault())
	importDef := ast.NewImport(tokStr, &tokIdent, common.SpanDefault())
	return ast.NewSemImport(*importDef, name, analysis)
}

type ProjectAnalysis struct {
	Main      string // path to main.gluax
	Config    frontend.GluaxToml
	Workspace string
	overrides map[string]string

	OsRoot *os.Root

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
		Workspace: workspace,
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

func (pa *ProjectAnalysis) newAnalysis(path string) *Analysis {
	scope := pa.currentState.RootScope
	if !pa.Config.Std || !strings.Contains(path, typesFile) {
		scope = pa.currentState.RootScope.Child(false)
	}
	return &Analysis{
		Workspace: pa.Workspace,
		Src:       path,
		Scope:     scope,
		Project:   pa,
		State:     pa.currentState,
	}
}

func (pa *ProjectAnalysis) parseFile(path string) (*Analysis, error) {
	analysis := pa.newAnalysis(path)

	// Load file content (override or from disk)
	code, err := pa.ReadFile(path)
	if err != nil {
		analysis.Errorf(common.SpanDefault(), "Failed to load file: %v", err)
		return analysis, fmt.Errorf("failed to load file: %w", err)
	}

	macros := map[string]string{
		pa.currentState.Label: "",
	}
	preprocessed, diag := preprocess.Preprocess(code, macros)
	if diag != nil {
		analysis.Diags = append(analysis.Diags, *diag)
		return analysis, fmt.Errorf("preprocessing failed")
	}

	toks, diag := lexer.Lex(path, preprocessed)
	if diag != nil {
		analysis.Diags = append(analysis.Diags, *diag)
		return analysis, fmt.Errorf("lexing failed")
	}

	astRoot, errors, hardErr := parser.Parse(toks)
	if hardErr {
		analysis.Diags = append(analysis.Diags, errors...)
		return analysis, fmt.Errorf("parsing failed")
	}

	if len(errors) > 0 {
		analysis.Diags = append(analysis.Diags, errors...)
	}

	analysis.Ast = astRoot
	return analysis, nil
}

// AnalyzeFromEntryPoint takes a single file path, discovers all its dependencies,
// runs the full multi-phase analysis on the entire graph
func (pa *ProjectAnalysis) AnalyzeFromEntryPoint(entryPointPath string) error {
	stateFiles := pa.getStateFiles()

	queue := []string{entryPointPath}

	filesInGraph := make([]*Analysis, 0, 10)

	// we use a loop instead of recursion to avoid stack overflows
	i := 0
	for i < len(queue) {
		path := queue[i]
		i++

		// if the file has already been processed in the current state then skip it
		if _, ok := stateFiles[path]; ok {
			continue
		}

		analysis, err := pa.parseFile(path)
		if err != nil {
			stateFiles[path] = analysis
			continue
		}

		stateFiles[path] = analysis
		filesInGraph = append(filesInGraph, analysis)

		// add this file's imports to the queue to be parsed.
		for _, imp := range analysis.Ast.Imports {
			resolvedPath, resolveErr := analysis.resolveImportPath(analysis.Src, imp.Path.Raw)
			if resolveErr != nil {
				analysis.Errorf(imp.Path.Span(), "import error: %v", resolveErr)
				continue
			}
			queue = append(queue, resolvedPath)
		}
	}

	runPhase := func(phaseFunc func(*Analysis)) {
		for _, analysis := range filesInGraph {
			defer func() {
				if r := recover(); r != nil {
					if errStr, ok := r.(string); ok {
						if errStr != "" {
							log.Printf("panic: %v\n%s", errStr, debug.Stack())
							analysis.Error(common.SpanDefault(), errStr)
						}
					} else {
						log.Printf("panic: %v\n%s", r, debug.Stack())
						analysis.Errorf(common.SpanDefault(), "%v", r)
					}
				}
			}()
			phaseFunc(analysis)
		}
	}

	runPhase(func(a *Analysis) {
		for _, imp := range a.Ast.Imports {
			a.handleImport(a.Scope, imp)
		}
	})
	runPhase(func(a *Analysis) { a.populateDeclarations() })
	runPhase(func(a *Analysis) { a.resolveUses() })
	runPhase(func(a *Analysis) { a.resolveImplementations() })
	runPhase(func(a *Analysis) { a.analyzeImplementations() })

	return nil
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
	oldWs, oldConfig, oldRootScope := pa.Workspace, pa.Config, pa.currentState.RootScope
	pa.Workspace = pkgPath

	mainPath := filepath.Join(pkgPath, "src", "main.gluax")

	isMain := pa.Workspace == oldWs

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
	pa.Main = mainPath

	if pa.Config.Std {
		builtinTypesFile := common.FilePathClean(filepath.Join(pa.Workspace, typesFile))
		pa.overrides[builtinTypesFile] = ast.BuiltinTypes
		if err := pa.AnalyzeFromEntryPoint(builtinTypesFile); err != nil {
			return fmt.Errorf("failed to analyze built-in types: %w", err)
		}
		delete(pa.overrides, builtinTypesFile)
	}

	if err := pa.AnalyzeFromEntryPoint(mainPath); err != nil {
		return fmt.Errorf("failed to analyze package starting from %s: %w", mainPath, err)
	}

	state := pa.currentState
	// don't leak full path
	// for p, a := range state.Files {
	// 	if !pa.StartsWithWorkspace(p) {
	// 		continue
	// 	}
	// 	delete(state.Files, p)
	// 	state.Files[pa.PathRelativeToWorkspace(p)] = a
	// }

	packageName := pa.CurrentPackage()

	pa.Workspace, pa.Config, pa.currentState.RootScope = oldWs, oldConfig, oldRootScope

	if !isMain {
		analysis := state.Files[mainPath]
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
		for name, symA := range publicAnalysis.Scope.Symbols {
			nameSyms := make([]*ast.Symbol, 0, len(symA))
			for _, sym := range symA {
				if sym.IsPublic() {
					nameSyms = append(nameSyms, sym)
				}
			}
			pa.currentState.RootScope.Symbols[name] = nameSyms
		}
	}
	if err := pa.processPackage(workspace, true); err != nil {
		return err
	}
	return nil
}

func AnalyzeProject(workspace string, overrides map[string]string) (*ProjectAnalysis, error) {
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
