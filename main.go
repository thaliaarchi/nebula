// Nebula is a compiler for stack-based languages targeting LLVM IR.
package main

import (
	"flag"
	"fmt"
	"go/token"
	"io/ioutil"
	"os"
	"strings"

	"github.com/andrewarchi/graph"
	"github.com/andrewarchi/nebula/ir"
	"github.com/andrewarchi/nebula/ir/codegen"
	"github.com/andrewarchi/nebula/ir/optimize"
	"github.com/andrewarchi/nebula/ws"
	"llvm.org/llvm/bindings/go/llvm"
)

var (
	name = os.Args[0]

	ascii           bool
	format          string
	noFold          bool
	maxStackLen     uint
	maxCallStackLen uint
	maxHeapBound    uint

	commands    map[string]commandConfig
	packFlags   = flag.NewFlagSet("pack", flag.ExitOnError)
	unpackFlags = flag.NewFlagSet("unpack", flag.ExitOnError)
	graphFlags  = flag.NewFlagSet("graph", flag.ExitOnError)
	astFlags    = flag.NewFlagSet("ast", flag.ExitOnError)
	irFlags     = flag.NewFlagSet("ir", flag.ExitOnError)
	llvmFlags   = flag.NewFlagSet("llvm", flag.ExitOnError)
	helpFlags   = flag.NewFlagSet("help", flag.ExitOnError)
)

type commandConfig struct {
	run   func([]string)
	flags *flag.FlagSet
}

const (
	usageText = `Nebula is a compiler for stack-based languages targeting LLVM IR.

Usage:

	%s <command> [arguments] <program>

The commands are:

	pack    compress program to bit packed format
	unpack  uncompress program from bit packed format
	graph   print Nebula IR control flow graph
	ast     emit Whitespace AST
	ir      emit Nebula IR
	llvm    emit LLVM IR

Use "%s help <command>" for more information about a command.

Examples:

	%s ir programs/pi.out.ws > pi.nir
	%s llvm programs/ascii4.out.ws > ascii4.ll
	%s llvm -heap=400000 programs/interpret.out.ws > interpret.ll
	%s graph programs/interpret.out.ws | dot -Tpng > graph.png

`
	packHeader   = "Pack compresses a program to the bit packed format."
	unpackHeader = "Unpack decompresses a program from the bit packed format."
	graphHeader  = "Graph prints the control flow graph of a program's Nebula IR."
	astHeader    = "AST emits a program's AST in Whitespace syntax."
	irHeader     = "IR emits the Nebula IR of a program."
	llvmHeader   = "LLVM emits the LLVM IR of a program."
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	initFlags()
	commandName := os.Args[1]
	command, ok := commands[commandName]
	if !ok {
		helpFlags.Parse(os.Args[1:]) // print usage if a help flag given
		usageErrorf("%s %s: unknown command", name, commandName)
	}
	command.flags.Parse(os.Args[2:])
	command.run(command.flags.Args())
}

func usage() {
	fmt.Fprintf(os.Stderr, usageText, name, name, name, name, name, name)
}

func initFlags() {
	commands = map[string]commandConfig{
		"pack":   {runPack, packFlags},
		"unpack": {runUnpack, unpackFlags},
		"graph":  {runGraph, graphFlags},
		"ast":    {runAST, astFlags},
		"ir":     {runIR, irFlags},
		"llvm":   {runLLVM, llvmFlags},
		"help":   {runHelp, helpFlags},
	}
	graphFlags.BoolVar(&ascii, "ascii", false, "print as ASCII grid rather than DOT digraph")
	astFlags.StringVar(&format, "format", "wsa", "output format; options: ws, wsa, wsx, wsapos, wsacomment")
	llvmFlags.UintVar(&maxStackLen, "stack", codegen.DefaultMaxStackLen, "maximum stack length for LLVM codegen")
	llvmFlags.UintVar(&maxCallStackLen, "calls", codegen.DefaultMaxCallStackLen, "maximum call stack length for LLVM codegen")
	llvmFlags.UintVar(&maxHeapBound, "heap", codegen.DefaultMaxHeapBound, "maximum heap address bound for LLVM codegen")
	addIRFlags(graphFlags)
	addIRFlags(irFlags)
	addIRFlags(llvmFlags)
	setUsage(packFlags, "pack <program>", packHeader, false)
	setUsage(unpackFlags, "unpack <program>", unpackHeader, false)
	setUsage(graphFlags, "graph [-ascii] [-nofold] <program>", graphHeader, true)
	setUsage(astFlags, "ast [-format=f] <program>", astHeader, true)
	setUsage(irFlags, "ir [-nofold] <program>", irHeader, true)
	setUsage(llvmFlags, "llvm [-nofold] [-stack=n] [-calls=n] [-heap=n] <program>", llvmHeader, true)
	helpFlags.Usage = usage
}

func addIRFlags(flags *flag.FlagSet) {
	flags.BoolVar(&noFold, "nofold", false, "disable constant folding")
}

func setUsage(flags *flag.FlagSet, usage, header string, printFlags bool) {
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s %s\n", name, usage)
		if header != "" {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, header)
		}
		if printFlags {
			fmt.Fprint(os.Stderr, "\nOptions:\n\n")
			flags.PrintDefaults()
		}
	}
}

func readFile(args []string) (string, []byte) {
	if len(args) == 0 {
		usageError("No program provided.")
	}
	if len(args) != 1 {
		usageError("Too many arguments provided.")
	}
	filename := args[0]
	src, err := ioutil.ReadFile(filename)
	if err != nil {
		exitError(err)
	}
	return filename, src
}

func lex(src []byte, filename string) *ws.Program {
	fset := token.NewFileSet()
	file := fset.AddFile(filename, -1, len(src))
	tokens, err := ws.LexTokens(file, src)
	if err != nil {
		exitError(err)
	}
	program := &ws.Program{File: file, Tokens: tokens}

	mapFilename := filename + ".map"
	if info, err := os.Stat(mapFilename); err == nil && !info.IsDir() {
		sourceMap, err := os.Open(mapFilename)
		if err != nil {
			exitError(err)
		}
		defer sourceMap.Close()
		program.LabelNames, err = ws.ParseSourceMap(sourceMap)
		if err != nil {
			exitError(err)
		}
	}
	return program
}

func lexFile(args []string) (*ws.Program, []byte) {
	filename, src := readFile(args)
	switch {
	case strings.HasSuffix(filename, ".wsa"):
		exitError("WSA lexing not implemented.")
	case strings.HasSuffix(filename, ".wsx"):
		src = ws.Unpack(src)
	}
	return lex(src, filename), src
}

func convertSSA(args []string) *ir.Program {
	program, _ := lexFile(args)
	ssa, errs := program.LowerIR()
	if len(errs) != 0 {
		fatal := false
		for _, err := range errs {
			if _, ok := err.(*ir.RetUnderflowError); !ok {
				fatal = true
			}
			fmt.Fprintln(os.Stderr, err)
		}
		if fatal {
			os.Exit(1)
		}
	}
	ssa.TrimUnreachable()
	if !noFold {
		optimize.FoldConstArith(ssa)
	}
	return ssa
}

func runPack(args []string) {
	filename, src := readFile(args)
	switch {
	case strings.HasSuffix(filename, ".wsa"):
		exitError("WSA lexing not implemented.")
	case strings.HasSuffix(filename, ".wsx"):
		usageError("Program is already packed.")
	}
	fmt.Print(string(ws.Pack(src)))
}

func runUnpack(args []string) {
	_, src := readFile(args)
	fmt.Print(string(ws.Unpack(src)))
}

func runGraph(args []string) {
	ssa := convertSSA(args)
	if !ascii {
		fmt.Print(ssa.DotDigraph())
	} else {
		labels := make([]string, len(ssa.Blocks))
		for i, block := range ssa.Blocks {
			labels[i] = block.Name()
		}
		fmt.Print(graph.FormatGridLabeled(optimize.ControlFlowGraph(ssa), labels))
	}
}

func runAST(args []string) {
	program, src := lexFile(args)
	switch format {
	case "ws":
		fmt.Print(program.DumpWS())
	case "wsa":
		fmt.Print(program.Dump("    "))
	case "wsx":
		fmt.Print(string(ws.Pack([]byte(program.DumpWS()))))
	case "wsapos":
		fmt.Print(program.DumpPos())
	case "wsacomment":
		fmt.Print(program.DumpCommented(src, "    "))
	default:
		exitErrorf("Unknown format: %s.", format)
	}
}

func runIR(args []string) {
	program := convertSSA(args)
	fmt.Print(program.String())
}

func runLLVM(args []string) {
	program := convertSSA(args)
	mod := codegen.EmitLLVMModule(program, codegen.Config{
		MaxStackLen:     maxStackLen,
		MaxCallStackLen: maxCallStackLen,
		MaxHeapBound:    maxHeapBound,
	})
	if err := llvm.VerifyModule(mod, llvm.PrintMessageAction); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	fmt.Print(mod.String())
}

func runHelp(args []string) {
	if len(args) == 1 {
		command, ok := commands[args[0]]
		if ok {
			command.flags.Usage()
			os.Exit(2)
		}
	}
	usage()
	os.Exit(2)
}

func exitError(msg interface{}) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func exitErrorf(format string, args ...interface{}) {
	exitError(fmt.Sprintf(format, args...))
}

func usageError(msg interface{}) {
	fmt.Fprintln(os.Stderr, msg)
	fmt.Fprintf(os.Stderr, "Run '%s help' for usage.\n", name)
	os.Exit(2)
}

func usageErrorf(format string, args ...interface{}) {
	usageError(fmt.Sprintf(format, args...))
}
