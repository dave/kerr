// +build !js

package kerr

import (
	"fmt"
	"os"
	"path/filepath"
)

func formatLocation(e Struct) string {
	return fmt.Sprintf(" in %s:%d %s", getRelPath(e.File), e.Line, e.Function)
}

func getRelPath(filePath string) string {
	wd, err := os.Getwd()
	if err != nil {
		// notest
		return filePath
	}
	out, err := filepath.Rel(wd, filePath)
	if err != nil {
		return filePath
	}
	return out
}
