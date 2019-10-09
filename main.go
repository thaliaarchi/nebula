package main

import (
	"fmt"
	"os"

	"github.com/andrewarchi/graph"
	"github.com/andrewarchi/wspace/ast"
	"github.com/andrewarchi/wspace/bigint"
	"github.com/andrewarchi/wspace/token"
	"github.com/andrewarchi/wspace/ws"
)

const usage = `wspace ast <file> [trim?]
wspace matrix <file> [trim?]
wspace dot <file> [trim?] | dot -Tpng > graph.png`

func main() {
	if len(os.Args) < 3 || len(os.Args) > 4 {
		fmt.Println(usage)
		return
	}
	mode := os.Args[1]
	if mode != "ast" && mode != "dot" && mode != "matrix" {
		fmt.Printf("Unrecognized mode: %s\n", mode)
		fmt.Println(usage)
		return
	}
	filename := os.Args[2]
	trim := len(os.Args) != 4 || os.Args[3] != "false"

	f, err := os.Open(filename)
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

	var labelNames *bigint.Map
	if _, err := os.Stat(filename + ".map"); err == nil {
		sourceMap, err := os.Open(filename + ".map")
		if err != nil {
			fmt.Println(err)
			return
		}
		labelNames, err = ws.ParseSourceMap(sourceMap)
	}

	ast, err := ast.Parse(tokens, labelNames, trim)
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
