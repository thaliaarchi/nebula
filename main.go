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
	instrChan := ws.Parse(l)
	var instrs []ws.Instr
	for instr := range instrChan {
		instrs = append(instrs, instr)
	}
	fmt.Println("Running:")
	vm, err := ws.NewVM(instrs)
	if err != nil {
		fmt.Println(err)
		return
	}
	vm.Run()
}
