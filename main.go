package main

import (
	"fmt"
	"os"

	"github.com/andrewarchi/graph"
	"github.com/andrewarchi/nebula/analysis"
	"github.com/andrewarchi/nebula/bigint"
	"github.com/andrewarchi/nebula/codegen"
	"github.com/andrewarchi/nebula/ir"
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

	var flagBit, flagDot, flagMatrix, flagIR, flagLLVM bool
	for _, mode := range os.Args[2:] {
		switch mode {
		case "bit":
			flagBit = true
		case "dot":
			flagDot = true
		case "matrix":
			flagMatrix = true
		case "ir":
			flagIR = true
		case "llvm":
			flagLLVM = true
		}
	}

	var r ws.SpaceReader
	if flagBit {
		r = ws.NewBitReader(f)
	} else {
		r = ws.NewTextReader(f)
	}
	tokens, err := ws.Lex(r)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
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

	p := ws.Program{Name: filename, Tokens: tokens, LabelNames: labelNames}
	program, err := p.ConvertSSA()
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

	if flagDot {
		fmt.Print(program.DotDigraph())
	}
	if flagMatrix {
		fmt.Print(graph.FormatMatrix(analysis.ControlFlowGraph(program)))
	}
	if flagIR {
		fmt.Print(program.String())
	}
	if flagLLVM {
		mod := codegen.EmitLLVMIR(program)
		if err := llvm.VerifyModule(mod, llvm.PrintMessageAction); err != nil {
			fmt.Fprintln(os.Stdout, err)
		}
		fmt.Print(mod.String())
	}
}
