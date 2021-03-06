package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/jpnock/see90/pkg/c90"
)

type CommentRemoverReader struct {
	strippingMultiline  bool
	strippingRestOfLine bool
	r                   io.Reader
	last                byte
}

func (r *CommentRemoverReader) Read(p []byte) (n int, err error) {
	tmp := make([]byte, len(p)+1)

	n, err = r.r.Read(tmp[1:])
	if err != nil {
		return n, err
	}

	if n == 0 {
		return 0, nil
	}

	tmp[0] = r.last

	for i := 0; i < n+1; i++ {
		c := tmp[i]
		var next byte
		if i+1 < n {
			next = tmp[i+1]
		}

		if r.strippingRestOfLine {
			if c == '\n' {
				r.strippingRestOfLine = false
			} else {
				tmp[i] = ' '
			}
		} else if r.strippingMultiline {
			if c == '*' && next == '/' {
				r.strippingMultiline = false
				tmp[i+1] = ' '
			}
			tmp[i] = ' '
		} else {
			if c == '/' && next == '*' {
				r.strippingMultiline = true
				tmp[i] = ' '
				tmp[i+1] = ' '
			} else if c == '/' && next == '/' {
				r.strippingRestOfLine = true
				tmp[i] = ' '
				tmp[i+1] = ' '
			}
		}
	}

	r.last = tmp[n-1]

	copy(p, tmp[1:])
	return n, nil
}

func main() {
	inputPath := flag.String("S", "test/all/main.c", "The input file path")
	outputPath := flag.String("o", "test/all/main.s", "The output file path")
	flag.Parse()

	inputFile, err := os.Open(*inputPath)
	if err != nil {
		log.Fatal(err)
	}
	outputFile, err := os.Create(*outputPath)
	if err != nil {
		log.Fatal(err)
	}

	c90.Parse(c90.NewLexer(&CommentRemoverReader{r: inputFile}))

	fmt.Fprint(os.Stderr, c90.AST.Describe(0))
	c90.AST.GenerateMIPS(outputFile, c90.NewMIPS())
}
