package main

import (
	"os"
	"path/filepath"
)

type NewCmd struct {
	Name string `arg:"" required:"" help:"Name of the new project."`
}

func (n *NewCmd) Run() error {
	projectDir := n.Name
	if err := os.MkdirAll(filepath.Join(projectDir, "src"), 0755); err != nil {
		return err
	}

	// .gitignore
	gitignoreContent := "out/\n"
	if err := os.WriteFile(filepath.Join(projectDir, ".gitignore"), []byte(gitignoreContent), 0644); err != nil {
		return err
	}

	// gluax.toml
	tomlContent := "name = \"" + n.Name + "\"\nversion = \"0.1\"\n"
	if err := os.WriteFile(filepath.Join(projectDir, "gluax.toml"), []byte(tomlContent), 0644); err != nil {
		return err
	}

	// src/main.gluax
	mainGluaxContent := "func main() {\n\n}\n"
	if err := os.WriteFile(filepath.Join(projectDir, "src", "main.gluax"), []byte(mainGluaxContent), 0644); err != nil {
		return err
	}

	return nil
}
