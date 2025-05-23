package sema

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/common"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

func (a *Analysis) resolveImportPath(currentFile, relative string) (string, error) {
	baseDir := filepath.Dir(currentFile)
	resolved := filepath.Join(baseDir, relative)
	if filepath.Ext(resolved) == "" {
		resolved += ".gluax"
	}
	resolved = common.FilePathClean(resolved)

	// Ensure inside <workspace>/src
	wsSrc := filepath.Join(a.Workspace, "src") + string(os.PathSeparator)
	if !strings.HasPrefix(resolved, common.FilePathClean(wsSrc)) {
		return "", fmt.Errorf("import path cannot be outside of `src` directory (got %s)", resolved)
	}

	// Must end with .gluax
	if filepath.Ext(resolved) != ".gluax" {
		return "", fmt.Errorf("file must be a .gluax file, not %s", resolved)
	}

	// if it's not inside overrides, then make sure file exists on disk
	if _, ok := a.Project.overrides[resolved]; !ok {
		if _, err := os.Stat(resolved); os.IsNotExist(err) {
			return "", fmt.Errorf("file does not exist: %s", resolved)
		}
	}

	return resolved, nil
}

func (a *Analysis) handleImport(scope *Scope, it *ast.Import) {
	importPath := it.Path.Raw
	if a.Project.Config.Std {
		if importPath == "globals*" {
			a.Project.importGlobals()
			return
		}
	}

	// Resolve relative path
	resolved, err := a.resolveImportPath(a.Src, importPath)
	if err != nil {
		a.Error(fmt.Sprintf("error importing file: %s", err), it.Path.Span())
		return
	}

	// Now analyze that file via the project:
	importedAnalysis, hardError := a.Project.AnalyzeFile(resolved)
	if hardError {
		a.Error("failed to import", it.Path.Span())
		return
	}

	if it.As == nil {
		inferred := strings.TrimSuffix(filepath.Base(resolved), filepath.Ext(resolved))
		if !lexer.IsValidIdent(inferred) {
			a.Error("file name is not a valid identifier to use, specify an alias", it.Path.Span())
			return
		}

		as := lexer.NewTokIdent(inferred, it.Path.Span())
		it.As = &as
	}

	importInfo := ast.NewSemImport(*it, resolved, importedAnalysis)
	if err := scope.AddImport(it.As.Raw, importInfo, it.As.Span(), it.Public); err != nil {
		a.Error(err.Error(), it.As.Span())
	}
}
