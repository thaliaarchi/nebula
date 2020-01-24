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
	"llvm.org/llvm/bindings/go/llvm"
)

const usage = `Usage:
	nebula [file] [modes...]
For example:
	nebula programs/interpret.out.ws dot | dot -Tpng > graph.png`

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, usage)
		return
	}

	filename := os.Args[1]
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
	// if mode != "llvm" {
	// 	analysis.FoldConstArith(program)
	// }
	// program.ConcatStrings() // not general

	modes := make(map[string]struct{})
	for _, mode := range os.Args[2:] {
		modes[mode] = struct{}{}
	}

	if _, ok := modes["dot"]; ok {
		fmt.Print(program.DotDigraph())
	}
	if _, ok := modes["matrix"]; ok {
		fmt.Print(graph.FormatMatrix(analysis.ControlFlowGraph(program)))
	}
	if _, ok := modes["ir"]; ok {
		fmt.Print(program.String())
	}
	if _, ok := modes["llvm"]; ok {
		mod := codegen.EmitLLVMIR(program)
		if err := llvm.VerifyModule(mod, llvm.PrintMessageAction); err != nil {
			fmt.Fprintln(os.Stdout, err)
		}
		fmt.Print(mod.String())
	}
}
