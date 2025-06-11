package std

import (
	"embed"
	"io/fs"
	"strings"

	"github.com/gluax-lang/gluax/common"
)

//go:embed std
var FS embed.FS

var Workspace string = "std"
var Files map[string]string = func() map[string]string {
	out := make(map[string]string)

	err := fs.WalkDir(FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil { // propagate unexpected I/O problems
			return err
		}
		if d.IsDir() { // nothing to read yet
			return nil
		}
		data, err := FS.ReadFile(p)
		if err != nil {
			return err
		}
		name := common.FilePathClean(strings.TrimPrefix(p, "./"))
		out[name] = string(data)
		return nil
	})
	if err != nil {
		panic(err)
	}
	return out
}()
