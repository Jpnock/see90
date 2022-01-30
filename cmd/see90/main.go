package main

import (
	"fmt"
	"os"

	"github.com/jpnock/see90/pkg/c90"
)

func main() {
	c90.Parse(c90.NewLexer(os.Stdin))
	fmt.Println(c90.AST.Describe())
}
