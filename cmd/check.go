package main

import (
	"path/filepath"

	"github.com/gluax-lang/gluax/frontend/sema"
)

type CheckCmd struct {
	Path string `help:"Path to the project directory." short:"p" default:"."`
}

func (c *CheckCmd) Run() error {
	absPath, err := filepath.Abs(c.Path)
	if err != nil {
		return err
	}

	{
		pAnalysis, err := sema.AnalyzeProject(absPath, map[string]string{})
		if err != nil {
			return err
		}

		for _, file := range pAnalysis.ServerFiles() {
			for _, diag := range file.Diags {
				println("SERVER", diag.Message)
			}
		}

		for _, file := range pAnalysis.ClientFiles() {
			for _, diag := range file.Diags {
				println("CLIENT", diag.Message)
			}
		}
	}

	return nil
}
