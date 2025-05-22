package main

import (
	"github.com/alecthomas/kong"
)

func main() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("gluax"),
		kong.Description("Gluax CLI"),
		kong.UsageOnError(),
	)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}

type CLI struct {
	Build   BuildCmd   `cmd:"" help:"Build the project." aliases:"compile"`
	New     NewCmd     `cmd:"" help:"Create a new project."`
	Lsp     LspCmd     `cmd:"" help:"Run the LSP server."`
	Version VersionCmd `cmd:"" help:"Show version."`
}
