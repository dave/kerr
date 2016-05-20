package kerr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetRelPath(t *testing.T) {
	wd, _ := os.Getwd()

	p := getRelPath("foo")
	if p != "foo" {
		t.Error()
	}

	p = getRelPath(filepath.Join(wd, "foo/bar"))
	if p != "foo/bar" {
		t.Error(p)
	}
}
