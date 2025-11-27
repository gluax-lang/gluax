package common

import (
	"net/url"
	"path/filepath"
	"runtime"
)

// FilePathClean is a combination of filepath.Clean and filepath.ToSlash
//
// Example:
//
//	C:\H\ -> C:/H
func FilePathClean(p string) string {
	// First do the normal OS-based cleanup
	cleaned := filepath.Clean(p)
	// Then normalize all separators to forward slash
	return filepath.ToSlash(cleaned)
}

func FilePathToURI(path string) string {
	u := &url.URL{
		Scheme: "file",
		Path:   path,
	}
	if runtime.GOOS == "windows" {
		u.Path = "/" + path // Windows needs extra leading slash
	}
	return u.String()
}
