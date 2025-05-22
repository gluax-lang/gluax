package main

import (
	"os"
	"path/filepath"
	"strings"

	codegen "github.com/gluax-lang/gluax/backend"
	"github.com/gluax-lang/gluax/frontend/sema"
)

type BuildCmd struct {
	Path string `help:"Path to the project directory." short:"p" default:"."`
}

func (b *BuildCmd) Run() error {
	absPath, err := filepath.Abs(b.Path)
	if err != nil {
		return err
	}

	pAnalysis, err := sema.AnalyzeProject(absPath, map[string]string{})
	if err != nil {
		return err
	}

	name := strings.ToLower(pAnalysis.Config.Name)

	outDir := filepath.Join(absPath, "out")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	serverCode, clientCode := codegen.GenerateProject(pAnalysis)

	svPath := filepath.Join(outDir, "sv_"+name+".lua")
	clPath := filepath.Join(outDir, "cl_"+name+".lua")

	if err := os.WriteFile(svPath, []byte(serverCode), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(clPath, []byte(clientCode), 0644); err != nil {
		return err
	}
	return nil
}
