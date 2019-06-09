package main

import (
	"fmt"
	"os"

	"github.com/andrewarchi/wspace/ws"
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
	l := ws.NewLexer(f)
	instrs := ws.Parse(l)
	for instr := range instrs {
		fmt.Printf("%s %d\n", &instr.Type, instr.Arg)
	}
}
