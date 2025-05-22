package main

import "github.com/gluax-lang/gluax/cmd/lsp"

type LspCmd struct {
	Stdio bool `help:"(internal) LSP clients pass this flag. Safe to ignore." name:"stdio"`
}

func (l *LspCmd) Run() error {
	return lsp.RunLSP()
}
