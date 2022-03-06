package main

import (
	"fmt"
	"os"

	"github.com/jpnock/see90/pkg/c90"
)

func main() {
	c90.Parse(c90.NewLexer(os.Stdin))
	fmt.Fprint(os.Stderr, c90.AST.Describe(0))
}
