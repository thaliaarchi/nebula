package main

import (
	"fmt"
	"os"

	"github.com/andrewarchi/graph"
	"github.com/andrewarchi/nebula/ast"
	"github.com/andrewarchi/nebula/bigint"
	"github.com/andrewarchi/nebula/token"
	"github.com/andrewarchi/nebula/ws"
)

const usage = `nebula ast <file>
nebula matrix <file>
nebula dot <file> | dot -Tpng > graph.png`

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
	filename := os.Args[2]

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

	ast, err := ast.Parse(tokens, labelNames)
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
