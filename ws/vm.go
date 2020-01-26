package ws // import "github.com/andrewarchi/nebula/ws"

/*
import (
	"bufio"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"

	"github.com/andrewarchi/nebula/ast"
	"github.com/andrewarchi/nebula/bigint"
	"github.com/andrewarchi/nebula/token"
)

const eofValue = 0
const debugHelp = `Commands:
  run
  continue
  step
  stepblock
  next
  quit
  print
  info
  help
`

type VM struct {
	entry *ast.BasicBlock
	block *ast.BasicBlock
	inst  int
	calls []*ast.BasicBlock
	stack bigint.Stack
	heap  bigint.Map
	in    *bufio.Reader
	out   *bufio.Writer
}

func NewVM(entry *ast.BasicBlock) (*VM, error) {
	return &VM{
		entry: entry,
		block: entry,
		inst:  0,
		calls: nil,
		stack: *bigint.NewStack(),
		heap:  *bigint.NewMap(func() interface{} { return new(big.Int) }),
		in:    bufio.NewReader(os.Stdin),
		out:   bufio.NewWriter(os.Stdout),
	}, nil
}

func (vm *VM) Run() {
	vm.Reset()
	vm.Continue()
}

func (vm *VM) Continue() {
	for !vm.Done() {
		vm.Step()
	}
	vm.out.Flush()
}

func (vm *VM) Step() {
	if vm.inst < len(vm.block.Tokens) {
		vm.Exec()
	} else {
		vm.ExecFlow()
	}
}

func (vm *VM) StepBlock() {
	for vm.inst < len(vm.block.Tokens) {
		vm.StepDebug()
	}
	vm.StepDebug()
}

func (vm *VM) StepDebug() {
	if vm.inst < len(vm.block.Tokens) {
		switch vm.block.Tokens[vm.inst].Type {
		case Printc, Printi:
			vm.out.WriteString(">> ")
			vm.Exec()
			vm.out.WriteByte('\n')
		case Readc, Readi:
			vm.out.WriteString("<< ")
			vm.Exec()
			vm.out.WriteByte('\n')
		default:
			vm.Exec()
		}
	} else {
		vm.ExecFlow()
	}
}

func (vm *VM) Next() {
	if vm.inst >= len(vm.block.Tokens) && vm.block.Flow.Type == Call {
		l := len(vm.calls)
		vm.StepBlock()
		for len(vm.calls) > l {
			vm.StepBlock()
		}
	} else {
		vm.StepDebug()
	}
}

func (vm *VM) Exit() {
	vm.block = nil
}

func (vm *VM) Done() bool {
	return vm.block == nil
}

func (vm *VM) Reset() {
	vm.block = vm.entry
	vm.inst = 0
	vm.calls = nil
	vm.stack.Clear()
	vm.heap.Clear()
}

func (vm *VM) Debug() {
	vm.Reset()
	for !vm.Done() {
		if vm.inst < len(vm.block.Tokens) {
			vm.out.WriteString(vm.block.Tokens[vm.inst].String())
		} else {
			vm.out.WriteString(vm.block.Flow.String())
		}
		vm.out.WriteByte('\n')
	prompt:
		vm.out.WriteString("(ws) ")
		vm.out.Flush()
		input, err := vm.in.ReadString('\n')
		if err != nil {
			vm.out.WriteString("Error: ")
			vm.out.WriteString(err.Error())
			vm.out.WriteByte('\n')
			break
		}
		input = strings.TrimSuffix(input, "\n")
		switch input {
		case "r", "run":
			vm.Run()
		case "c", "continue":
			vm.Continue()
		case "s", "step":
			vm.StepDebug()
		case "sb", "stepblock":
			vm.StepBlock()
		case "n", "next":
			vm.Next()
		case "q", "quit":
			vm.Exit()
		case "p", "print":
			vm.PrintBlock()
			goto prompt
		case "i", "info":
			vm.PrintInfo()
			goto prompt
		case "h", "help":
			vm.Help()
			goto prompt
		case "":
			goto prompt
		default:
			vm.out.WriteString("Unrecognized command: ")
			vm.out.WriteString(input)
			vm.out.WriteByte('\n')
			goto prompt
		}
	}
	vm.out.WriteString("-----\n")
	vm.PrintInfo()
	vm.out.Flush()
}

func (vm *VM) PrintBlock() {
	vm.out.WriteString(vm.block.Display())
	vm.out.WriteByte('\n')
}

func (vm *VM) PrintInfo() {
	vm.out.WriteString("Stack: ")
	vm.out.WriteString(vm.stack.String())
	vm.out.WriteString("\nHeap: ")
	vm.out.WriteString(vm.heap.String())
	vm.out.WriteByte('\n')
}

func (vm *VM) PrintStackTrace() {
	vm.out.WriteString(vm.block.Display())
	vm.out.WriteString("\n-----\n")
	vm.PrintInfo()
	vm.out.Flush()
}

func (vm *VM) Help() {
	vm.out.WriteString(debugHelp)
}

func (vm *VM) Exec() {
	inst := vm.block.Tokens[vm.inst]
	switch inst.Type {
	case Push:
		vm.stack.Push(inst.Arg)
	case Dup:
		vm.stack.Push(vm.stack.Top())
	case Copy:
		n, ok := bigint.ToInt(inst.Arg)
		if !ok {
			panic(fmt.Sprintf("copy argument overflow: %s", inst.Arg))
		}
		vm.stack.Push(vm.stack.Get(n))
	case Swap:
		vm.stack.Swap()
	case Drop:
		vm.stack.Pop()
	case Slide:
		n, ok := bigint.ToInt(inst.Arg)
		if !ok {
			panic(fmt.Sprintf("slide argument overflow: %s", inst.Arg))
		}
		vm.stack.Slide(n)

	case Add:
		vm.arith((*big.Int).Add)
	case Sub:
		vm.arith((*big.Int).Sub)
	case Mul:
		vm.arith((*big.Int).Mul)
	case Div:
		vm.arith((*big.Int).Div)
	case Mod:
		vm.arith((*big.Int).Mod)

	case Store:
		val, addr := vm.stack.Pop(), vm.stack.Pop()
		vm.heap.Retrieve(addr).(*big.Int).Set(val)
	case Retrieve:
		top := vm.stack.Top()
		top.Set(vm.heap.Retrieve(top).(*big.Int))

	case Printc:
		vm.out.WriteRune(bigint.ToRune(vm.stack.Pop()))
	case Printi:
		vm.out.WriteString(vm.stack.Pop().String())
	case Readc:
		vm.readRune(vm.heap.Retrieve(vm.stack.Pop()).(*big.Int))
	case Readi:
		vm.readInt(vm.heap.Retrieve(vm.stack.Pop()).(*big.Int))

	default:
		panic(fmt.Sprintf("unexpected instruction: %v", inst))
	}
	vm.inst++
}

func (vm *VM) ExecFlow() {
	switch vm.block.Flow.Type {
	case Call:
		vm.calls = append(vm.calls, vm.block.Next)
		vm.block = vm.block.Branch
	case Jmp:
		vm.block = vm.block.Branch
	case Jz:
		vm.block = vm.jmpSign(0)
	case Jn:
		vm.block = vm.jmpSign(-1)
	case Ret:
		if len(vm.calls) == 0 {
			panic("call stack underflow: ret")
		}
		vm.block = vm.calls[len(vm.calls)-1]
		vm.calls = vm.calls[:len(vm.calls)-1]
	case End:
		vm.block = nil
		vm.out.Flush()
	}
	vm.inst = 0
}

func (vm *VM) arith(op func(z, x, y *big.Int) *big.Int) {
	y, x := vm.stack.Pop(), vm.stack.Top()
	op(x, x, y)
}

func (vm *VM) jmpSign(sign int) *ast.BasicBlock {
	if vm.stack.Pop().Sign() == sign {
		return vm.block.Branch
	}
	return vm.block.Next
}

func (vm *VM) jmpCmp(cmp int, val *big.Int) *ast.BasicBlock {
	if vm.stack.Pop().Cmp(val) == cmp {
		return vm.block.Branch
	}
	return vm.block.Next
}

func (vm *VM) readRune(x *big.Int) {
	vm.out.Flush()
	r, _, err := vm.in.ReadRune()
	if err == io.EOF {
		x.SetInt64(eofValue)
		return
	}
	if err != nil {
		panic(fmt.Sprintf("readc: %v", err))
	}
	x.SetInt64(int64(r))
}

func (vm *VM) readInt(x *big.Int) {
	vm.out.Flush()
	line, err := vm.in.ReadString('\n')
	if err == io.EOF {
		x.SetInt64(eofValue)
		return
	}
	if err != nil {
		panic(fmt.Sprintf("readi: %v", err))
	}
	line = strings.TrimSuffix(line, "\n")
	x, ok := x.SetString(line, 10)
	if !ok {
		panic(fmt.Sprintf("invalid number: %v", line))
	}
}
*/
