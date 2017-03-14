package main

// ke: {"package": {"notest": true}}

import (
	"bytes"
	"flag"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	"github.com/dave/kerr"
	"github.com/dave/kerr/ksrc"
	"golang.org/x/tools/imports"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	flag.Parse()
	if err := processFile(flag.Arg(0)); err != nil {
		panic(err.Error())
	}
}

func processFile(filename string) error {

	if filename == "" {
		return kerr.New("CKBKFTNVXI", "File not found")
	}

	f, err := os.Open(filename)
	if err != nil {
		return kerr.Wrap("WCQEVUWMAI", err)
	}
	defer f.Close()

	src, err := ioutil.ReadAll(f)
	if err != nil {
		return kerr.Wrap("DWTHNAVSVY", err)
	}

	var res []byte

	res, err = ksrc.Process(filename, src)
	if err != nil {
		return kerr.Wrap("UFVVOEXSYE", err)
	}

	res, err = imports.Process(filename, res, nil)
	if err != nil {
		return kerr.Wrap("XFILQRWHSI", err)
	}

	if !bytes.Equal(src, res) {
		// formatting has changed
		err = ioutil.WriteFile(filename, res, 0)
		if err != nil {
			return kerr.Wrap("NFFAOHJQLG", err)
		}
	}

	return nil

}
