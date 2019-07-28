package main

import (
	"fmt"
	"os"

	"github.com/andrewarchi/wspace/ws"
)

const usage = `wspace run [file]
wspace debug [file]`

func main() {
	if len(os.Args) != 3 {
		fmt.Println(usage)
		return
	}
	mode := os.Args[1]
	if mode != "run" && mode != "debug" {
		fmt.Println("unrecognized mode")
		fmt.Println(usage)
		return
	}
	f, err := os.Open(os.Args[2])
	if err != nil {
		fmt.Println(err)
		return
	}

	r := ws.NewTextReader(f)
	tokenChan := ws.Lex(r)
	var tokens []ws.Token
	for token := range tokenChan {
		tokens = append(tokens, token)
	}
	ast, err := ws.NewAST(tokens)
	if err != nil {
		fmt.Println(err)
		return
	}
	vm, err := ws.NewVM(ast)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("ERROR: %v\n", r)
			vm.PrintStackTrace()
			os.Exit(1)
		}
	}()

	if mode == "run" {
		vm.Run()
	} else {
		vm.Debug()
	}
}
