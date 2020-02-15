package main

import (
	"flag"
	"fmt"
	"go/token"
	"io/ioutil"
	"os"
	"strings"

	"github.com/andrewarchi/graph"
	"github.com/andrewarchi/nebula/analysis"
	"github.com/andrewarchi/nebula/codegen"
	"github.com/andrewarchi/nebula/ir"
	"github.com/andrewarchi/nebula/ws"
	"llvm.org/llvm/bindings/go/llvm"
)

var (
	name     string
	flags    = flag.CommandLine
	commands = map[string]func([]string){
		"pack":   runPack,
		"unpack": runUnpack,
		"graph":  runGraph,
		"ast":    runAST,
		"ir":     runIR,
		"llvm":   runLLVM,
		"help":   runHelp,
	}
)

const usageText = `Nebula is a compiler for stack-based languages targeting LLVM IR.

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

func main() {
	flags.Usage = usage
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	name = os.Args[0]
	command := os.Args[1]
	commandFn, ok := commands[command]
	if !ok {
		usageErrorf("Unrecognized command: %s\n", command)
	}
	commandFn(os.Args[2:])
}

func usage() {
	fmt.Fprintf(flags.Output(), usageText, name, name, name, name, name, name)
	flags.PrintDefaults()
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
	lexer := ws.NewLexer(file, src)
	program, err := lexer.LexProgram()
	if err != nil {
		exitError(err)
	}

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

func lexFile(args []string) *ws.Program {
	filename, src := readFile(args)
	switch {
	case strings.HasSuffix(filename, ".wsa"):
		exitError("WSA lexing not implemented.")
	case strings.HasSuffix(filename, ".wsx"):
		src = ws.Unpack(src)
	}
	return lex(src, filename)
}

func convertSSA(p *ws.Program, noFold bool) *ir.Program {
	program, err := p.ConvertSSA()
	if err != nil {
		if _, ok := err.(*ir.ErrorRetUnderflow); !ok {
			exitError(err)
		}
		fmt.Fprintln(os.Stderr, err)
	}
	if !noFold {
		analysis.FoldConstArith(program)
	}
	return program
}

func runPack(args []string) {
	filename, src := readFile(args)
	if strings.HasSuffix(filename, ".wsa") {
		exitError("WSA lexing not implemented.")
	}
	fmt.Print(string(ws.Pack(src)))
}

func runUnpack(args []string) {
	_, src := readFile(args)
	fmt.Print(string(ws.Unpack(src)))
}

func runGraph(args []string) {
	ascii := *flags.Bool("ascii", false, "print as ASCII grid rather than DOT digraph.")
	noFold := irFlags()
	flags.Parse(args)

	program := convertSSA(lexFile(flags.Args()), noFold)
	if !ascii {
		fmt.Print(program.DotDigraph())
	} else {
		labels := make([]string, len(program.Blocks))
		for i, block := range program.Blocks {
			labels[i] = block.Name()
		}
		fmt.Print(graph.FormatGridLabeled(analysis.ControlFlowGraph(program), labels))
	}
}

func runAST(args []string) {
	format := *flags.String("format", "wsa", "output format; options: ws, wsa, wsx")
	flags.Parse(args)
	program := lexFile(args)
	switch format {
	case "ws":
		fmt.Print(program.DumpWS())
	case "wsa":
		fmt.Print(program.Dump("    "))
	case "wsx":
		fmt.Print(string(ws.Pack([]byte(program.DumpWS()))))
	default:
		exitErrorf("Unrecognized format: %s", format)
	}
}

func runIR(args []string) {
	noFold := irFlags()
	flags.Parse(args)

	program := convertSSA(lexFile(flags.Args()), noFold)
	fmt.Print(program.String())
}

func runLLVM(args []string) {
	maxStackLen := *flags.Uint("stack", codegen.DefaultMaxStackLen, "maximum stack length for LLVM codegen")
	maxCallStackLen := *flags.Uint("calls", codegen.DefaultMaxCallStackLen, "maximum call stack length for LLVM codegen")
	maxHeapBound := *flags.Uint("heap", codegen.DefaultMaxHeapBound, "maximum heap address bound for LLVM codegen")
	noFold := irFlags()
	flags.Parse(args)

	program := convertSSA(lexFile(flags.Args()), noFold)
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
	usage()
}

func irFlags() bool {
	noFold := *flags.Bool("nofold", false, "disable constant folding")
	return noFold
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
	fmt.Fprintf(os.Stderr, "Run %s help for usage.\n", name)
	os.Exit(2)
}

func usageErrorf(format string, args ...interface{}) {
	usageError(fmt.Sprintf(format, args...))
}
