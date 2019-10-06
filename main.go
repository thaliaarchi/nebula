package main

import (
	"fmt"
	"os"

	"github.com/andrewarchi/wspace/ast"
	"github.com/andrewarchi/wspace/token"
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
		fmt.Printf("Unrecognized mode: %s\n", mode)
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
	var tokens []token.Token
	for tok := range tokenChan {
		tokens = append(tokens, tok)
	}
	ast, err := ast.Parse(tokens)
	if err != nil {
		fmt.Println(err)
		return
	}
	ast.FoldConstArith()
	ast.ConcatStrings()
	fmt.Println(ast.String())
	// vm, err := ws.NewVM(ast)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	// defer func() {
	// 	if r := recover(); r != nil {
	// 		fmt.Printf("Error: %v\n", r)
	// 		vm.PrintStackTrace()
	// 		os.Exit(1)
	// 	}
	// }()

	// if mode == "run" {
	// 	vm.Run()
	// } else {
	// 	vm.Debug()
	// }
}
