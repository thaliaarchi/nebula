package ws

import (
	"bufio"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"

	"github.com/andrewarchi/wspace/ast"
	"github.com/andrewarchi/wspace/bigint"
	"github.com/andrewarchi/wspace/token"
)

const eofValue = 0
const debugHelp = `Commands:
  run
  continue
  step
  next
  quit
  info
  help
`

type VM struct {
	entry   *ast.Node
	inst    *ast.Node
	callers []*ast.Node
	stack   bigint.Stack
	heap    bigint.Map
	in      *bufio.Reader
	out     *bufio.Writer
}

func NewVM(entry *ast.Node) (*VM, error) {
	return &VM{
		entry:   entry,
		inst:    entry,
		callers: nil,
		stack:   *bigint.NewStack(),
		heap:    *bigint.NewMap(func() interface{} { return new(big.Int) }),
		in:      bufio.NewReader(os.Stdin),
		out:     bufio.NewWriter(os.Stdout),
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
	vm.Exec(vm.inst)
}

func (vm *VM) StepDebug() {
	switch vm.inst.Type {
	case token.Printc, token.Printi:
		vm.out.WriteString(">> ")
		vm.Step()
		vm.out.WriteByte('\n')
	case token.Readc, token.Readi:
		vm.out.WriteString("<< ")
		vm.Step()
		vm.out.WriteByte('\n')
	default:
		vm.Step()
	}
}

func (vm *VM) Next() {
	isCall := vm.inst.Type == token.Call
	vm.StepDebug()
	if isCall {
		for !vm.Done() {
			isRet := vm.inst.Type == token.Ret
			vm.StepDebug()
			if isRet {
				break
			}
		}
	}
}

func (vm *VM) Exit() {
	vm.inst = nil
}

func (vm *VM) Done() bool {
	return vm.inst == nil
}

func (vm *VM) Reset() {
	vm.inst = vm.entry
	vm.stack.Clear()
	vm.heap.Clear()
}

func (vm *VM) Debug() {
	vm.Reset()
	for !vm.Done() {
		vm.out.WriteString(vm.inst.Display())
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
		case "n", "next":
			vm.Next()
		case "q", "quit":
			vm.Exit()
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

func (vm *VM) PrintInfo() {
	vm.out.WriteString("Stack: ")
	vm.out.WriteString(vm.stack.String())
	vm.out.WriteString("\nHeap: ")
	vm.out.WriteString(vm.heap.String())
	vm.out.WriteByte('\n')
}

func (vm *VM) PrintStackTrace() {
	vm.out.WriteString(vm.inst.Display())
	vm.out.WriteString("\n-----\n")
	vm.PrintInfo()
	vm.out.Flush()
}

func (vm *VM) Help() {
	vm.out.WriteString(debugHelp)
}

func (vm *VM) Exec(inst *ast.Node) {
	next := inst.Next
	switch inst.Type {
	case token.Push:
		vm.stack.Push(inst.Arg)
	case token.Dup:
		vm.stack.Push(vm.stack.Top())
	case token.Copy:
		n, ok := bigint.ToInt(inst.Arg)
		if !ok {
			panic(fmt.Sprintf("copy argument overflow: %s", inst.Arg))
		}
		vm.stack.Push(vm.stack.Get(n))
	case token.Swap:
		vm.stack.Swap()
	case token.Drop:
		vm.stack.Pop()
	case token.Slide:
		n, ok := bigint.ToInt(inst.Arg)
		if !ok {
			panic(fmt.Sprintf("slide argument overflow: %s", inst.Arg))
		}
		vm.stack.Slide(n)

	case token.Add:
		vm.arith((*big.Int).Add)
	case token.Sub:
		vm.arith((*big.Int).Sub)
	case token.Mul:
		vm.arith((*big.Int).Mul)
	case token.Div:
		vm.arith((*big.Int).Div)
	case token.Mod:
		vm.arith((*big.Int).Mod)

	case token.Store:
		val, addr := vm.stack.Pop(), vm.stack.Pop()
		vm.heap.Retrieve(addr).(*big.Int).Set(val)
	case token.Retrieve:
		top := vm.stack.Top()
		top.Set(vm.heap.Retrieve(top).(*big.Int))

	case token.Call:
		vm.callers = append(vm.callers, inst.Next)
		next = inst.Branch
	case token.Jmp:
		next = inst.Branch
	case token.Jz:
		next = vm.jmpSign(0, inst)
	case token.Jn:
		next = vm.jmpSign(-1, inst)
	case token.Ret:
		if len(vm.callers) == 0 {
			panic("call stack underflow: ret")
		}
		next = vm.callers[len(vm.callers)-1]
		vm.callers = vm.callers[:len(vm.callers)-1]
	case token.End:
		next = nil
		vm.out.Flush()

	case token.Printc:
		vm.out.WriteRune(bigint.ToRune(vm.stack.Pop()))
	case token.Printi:
		vm.out.WriteString(vm.stack.Pop().String())
	case token.Readc:
		vm.readRune(vm.heap.Retrieve(vm.stack.Pop()).(*big.Int))
	case token.Readi:
		vm.readInt(vm.heap.Retrieve(vm.stack.Pop()).(*big.Int))
	}
	vm.inst = next
}

func (vm *VM) arith(op func(z, x, y *big.Int) *big.Int) {
	y, x := vm.stack.Pop(), vm.stack.Top()
	op(x, x, y)
}

func (vm *VM) jmpSign(sign int, inst *ast.Node) *ast.Node {
	if vm.stack.Pop().Sign() == sign {
		return inst.Branch
	}
	return inst.Next
}

func (vm *VM) jmpCmp(cmp int, inst *ast.Node, val *big.Int) *ast.Node {
	if vm.stack.Pop().Cmp(val) == cmp {
		return inst.Branch
	}
	return inst.Next
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
