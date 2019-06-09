package main

import (
	"fmt"
	"os"

	"github.com/andrewarchi/wspace"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("wspace [file]")
		return
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}
	l := wspace.NewLexer(f)
	instrs := wspace.Parse(l)
	for instr := range instrs {
		fmt.Printf("%s %d\n", &instr.Type, instr.Arg)
	}
}
