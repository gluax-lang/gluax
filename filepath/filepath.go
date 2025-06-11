package file_path

import (
	"net/url"
	"path/filepath"
	"runtime"
)

// FilePathClean is a combination of filepath.Clean and filepath.ToSlash
//
// Example:
//   C:\H\ -> C:/H
func Clean(p string) string {
	// First do the normal OS-based cleanup
	cleaned := filepath.Clean(p)
	// Then normalize all separators to forward slash
	return filepath.ToSlash(cleaned)
}

func ToURI(path string) string {
	uri := path
	if runtime.GOOS == "windows" {
		// Windows file URIs need three slashes: file:///C:/path
		uri = "file:///" + url.PathEscape(uri)
	} else {
		// Unix-like systems: file:///path
		uri = "file://" + url.PathEscape(uri)
	}
	return uri
}
