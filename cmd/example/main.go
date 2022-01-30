package main

import (
	"fmt"
	"os"

	"github.com/jpnock/see90/internal/firsttest"
)

func main() {
	firsttest.Parse(firsttest.NewLexer(os.Stdin))

	for i, val := range firsttest.RootTree {
		fmt.Printf("line %d : calculated %d\n", i+1, val)
	}
}
