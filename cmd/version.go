package main

import (
	"fmt"
)

var Version = "dev" // replaced by linker flag at build time

type VersionCmd struct{}

func (v *VersionCmd) Run() error {
	fmt.Println("gluax version:", Version)
	return nil
}
