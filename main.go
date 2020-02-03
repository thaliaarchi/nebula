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

	var bitPacked, fold, emitWs, emitWsa, emitDot, emitMatrix, emitIR, emitLLVM bool
	for _, mode := range os.Args[2:] {
		switch mode {
		case "bit":
			bitPacked = true
		case "fold":
			fold = true
		case "ws":
			emitWs = true
		case "wsa":
			emitWsa = true
		case "dot":
			emitDot = true
		case "matrix":
			emitMatrix = true
		case "ir":
			emitIR = true
		case "llvm":
			emitLLVM = true
		}
	}

	var r ws.SpaceReader
	if bitPacked {
		r = ws.NewBitReader(f, filename)
	} else {
		r = ws.NewTextReader(f, filename)
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
	if emitWs {
		fmt.Print(p.DumpWS())
	}
	if emitWsa {
		fmt.Print(p.Dump("    "))
	}

	program, err := p.ConvertSSA()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if _, ok := err.(*ir.ErrorRetUnderflow); !ok {
			return
		}
	}

	// program.JoinSimpleEntries() // incorrect
	if fold {
		analysis.FoldConstArith(program)
	}
	// program.ConcatStrings() // not general

	if emitDot {
		fmt.Print(program.DotDigraph())
	}
	if emitMatrix {
		fmt.Print(graph.FormatMatrix(analysis.ControlFlowGraph(program)))
	}
	if emitIR {
		fmt.Print(program.String())
	}
	if emitLLVM {
		mod := codegen.EmitLLVMIR(program)
		if err := llvm.VerifyModule(mod, llvm.PrintMessageAction); err != nil {
			fmt.Fprintln(os.Stdout, err)
		}
		fmt.Print(mod.String())
	}
}
