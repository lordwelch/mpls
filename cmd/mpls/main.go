package main

import (
	"io"
	"os"
	"path/filepath"

	"timmy.narnian.us/mpls"

	"github.com/kr/pretty"
)

func main() {
	var (
		file io.Reader
		Mpls mpls.MPLS
		err  error
	)
	file, err = os.Open(filepath.Clean(os.Args[1]))
	if err != nil {
		panic(err)
	}

	Mpls, err = mpls.Parse(file)
	pretty.Println(Mpls)
	if err != nil {
		panic(err)
	}
}
