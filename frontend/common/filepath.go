package common

import "path/filepath"

// FilePathClean is a combination of filepath.Clean and filepath.ToSlash
//
// Example:
//   C:\H\ -> C:/H
func FilePathClean(p string) string {
	// First do the normal OS-based cleanup
	cleaned := filepath.Clean(p)
	// Then normalize all separators to forward slash
	return filepath.ToSlash(cleaned)
}
