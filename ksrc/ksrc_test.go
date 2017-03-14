package ksrc

import (
	"testing"

	"regexp"

	"github.com/dave/ktest/assert"
	"github.com/dave/ktest/require"
)

func TestSource(t *testing.T) {

	_, err := Process("/foo.go", []byte("foo"))
	assert.IsError(t, err, "UIXBVGAPWL")

	in := `package ksrc

import "github.com/dave/kerr"

func foo() {
	var err error
	kerr.New()
	kerr.New("foo", "bar")
	kerr.Wrap(err)
}
`
	out, err := Process("/foo.go", []byte(in))
	require.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(`kerr\.New\(\"[A-Z]{10}\"\)\n`), string(out))
	assert.Regexp(t, regexp.MustCompile(`kerr\.New\(\"[A-Z]{10}\", \"foo\", \"bar\"\)\n`), string(out))
	assert.Regexp(t, regexp.MustCompile(`kerr\.Wrap\(\"[A-Z]{10}\", err\)\n`), string(out))

	in = `package ksrc

import "github.com/dave/kerr"

func foo() {
	fmt.Sprint("")
}
`
	out, err = Process("/foo.go", []byte(in))
	require.NoError(t, err)
	assert.Equal(t, in, string(out))

}
