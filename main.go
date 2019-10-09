package main

import (
	"fmt"
	"os"

	"github.com/andrewarchi/graph"
	"github.com/andrewarchi/wspace/ast"
	"github.com/andrewarchi/wspace/token"
	"github.com/andrewarchi/wspace/ws"
)

const usage = `wspace ast [file]
wspace matrix [file]
wspace dot [file] | dot -Tpng > graph.png`

func main() {
	if len(os.Args) != 3 {
		fmt.Println(usage)
		return
	}
	mode := os.Args[1]
	if mode != "ast" && mode != "dot" && mode != "matrix" {
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

	switch mode {
	case "ast":
		fmt.Print(ast.String())
	case "matrix":
		fmt.Print(graph.FormatMatrix(ast.ControlFlowGraph()))
	case "dot":
		fmt.Print(ast.DotDigraph())
	}

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
