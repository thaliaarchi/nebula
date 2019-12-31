package main

import (
	"fmt"
	"os"

	"github.com/andrewarchi/graph"
	"github.com/andrewarchi/nebula/analysis"
	"github.com/andrewarchi/nebula/bigint"
	"github.com/andrewarchi/nebula/codegen"
	"github.com/andrewarchi/nebula/ir"
	"github.com/andrewarchi/nebula/token"
	"github.com/andrewarchi/nebula/ws"
)

const usage = `nebula ir <file>
nebula matrix <file>
nebula dot <file> | dot -Tpng > graph.png`

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, usage)
		return
	}
	mode := os.Args[1]
	if mode != "ir" && mode != "dot" && mode != "matrix" && mode != "llvm" {
		fmt.Fprintf(os.Stderr, "Unrecognized mode: %s\n", mode)
		fmt.Fprintln(os.Stderr, usage)
		return
	}

	filename := os.Args[2]
	f, err := os.Open(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
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
			fmt.Fprintln(os.Stderr, err)
			return
		}
		labelNames, err = ws.ParseSourceMap(sourceMap)
	}

	program, err := ir.Parse(tokens, labelNames, filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if _, ok := err.(*ir.ErrorRetUnderflow); !ok {
			return
		}
	}

	// program.JoinSimpleEntries() // incorrect
	if mode != "llvm" {
		analysis.FoldConstArith(program)
	}
	// program.ConcatStrings() // not general

	switch mode {
	case "ir":
		fmt.Print(program.String())
	case "dot":
		fmt.Print(program.DotDigraph())
	case "matrix":
		fmt.Print(graph.FormatMatrix(analysis.ControlFlowGraph(program)))
	case "llvm":
		fmt.Println("LLVM IR:")
		codegen.EmitLLVMIR(program)
		fmt.Println("Nebula IR:")
		fmt.Print(program.String())
	}

	// vm, err := ws.NewVM(program)
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
