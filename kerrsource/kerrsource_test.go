package kerrsource

import (
	"testing"

	"regexp"

	"kego.io/kerr/assert"
)

func TestSource(t *testing.T) {

	_, err := Process("/foo.go", []byte("foo"))
	assert.IsError(t, err, "UIXBVGAPWL")

	in := `package kerrsource

import "kego.io/kerr"

func foo() {
	var err error
	kerr.New()
	kerr.New("foo", "bar")
	kerr.Wrap(err)
}
`
	out, err := Process("/foo.go", []byte(in))
	assert.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(`kerr\.New\(\"[A-Z]{10}\"\)\n`), string(out))
	assert.Regexp(t, regexp.MustCompile(`kerr\.New\(\"[A-Z]{10}\", \"foo\", \"bar\"\)\n`), string(out))
	assert.Regexp(t, regexp.MustCompile(`kerr\.Wrap\(\"[A-Z]{10}\", err\)\n`), string(out))

	in = `package kerrsource

import "kego.io/kerr"

func foo() {
	fmt.Sprint("")
}
`
	out, err = Process("/foo.go", []byte(in))
	assert.NoError(t, err)
	assert.Equal(t, in, string(out))

}
