// Package codegen lowers Nebula IR to LLVM IR.
//
package codegen // import "github.com/andrewarchi/nebula/ir/codegen"

import (
	"fmt"

	"github.com/andrewarchi/nebula/internal/bigint"
	"github.com/andrewarchi/nebula/ir"
	"llvm.org/llvm/bindings/go/llvm"
)

type moduleBuilder struct {
	ctx    llvm.Context
	b      llvm.Builder
	module llvm.Module
	config Config

	program    *ir.Program
	blocks     map[*ir.BasicBlock]llvm.BasicBlock
	blockNames map[*ir.BasicBlock]llvm.Value
	defs       map[ir.Value]llvm.Value

	stack        llvm.Value
	stackLen     llvm.Value
	callStack    llvm.Value
	callStackLen llvm.Value
	heap         llvm.Value

	main           llvm.Value
	printc         llvm.Value
	printi         llvm.Value
	readc          llvm.Value
	readi          llvm.Value
	flush          llvm.Value
	checkStack     llvm.Value
	checkCallStack llvm.Value
}

// Config contains allocation size configuration for codegen.
type Config struct {
	MaxStackLen     uint
	MaxCallStackLen uint
	MaxHeapBound    uint
}

// Default configuration values.
const (
	DefaultMaxStackLen     = 1024
	DefaultMaxCallStackLen = 256
	DefaultMaxHeapBound    = 4096
)

var (
	zero = llvm.ConstInt(llvm.Int64Type(), 0, false)
	one  = llvm.ConstInt(llvm.Int64Type(), 1, false)
)

// EmitLLVMModule generates a LLVM IR module for the given program.
func EmitLLVMModule(program *ir.Program, config Config) llvm.Module {
	ctx := llvm.GlobalContext()
	m := moduleBuilder{
		ctx:        ctx,
		b:          ctx.NewBuilder(),
		module:     ctx.NewModule(program.Name),
		config:     config,
		program:    program,
		blocks:     make(map[*ir.BasicBlock]llvm.BasicBlock),
		blockNames: make(map[*ir.BasicBlock]llvm.Value),
		defs:       make(map[ir.Value]llvm.Value),
	}
	m.declareFuncs()
	m.declareGlobals()
	m.emitBlocks()
	return m.module
}

func (m *moduleBuilder) declareFuncs() {
	mainTyp := llvm.FunctionType(llvm.Int32Type(), []llvm.Type{}, false)
	m.main = llvm.AddFunction(m.module, "main", mainTyp)

	printcTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int64Type()}, false)
	printiTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int64Type()}, false)
	readcTyp := llvm.FunctionType(llvm.Int64Type(), []llvm.Type{}, false)
	readiTyp := llvm.FunctionType(llvm.Int64Type(), []llvm.Type{}, false)
	flushTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{}, false)
	cStrTyp := llvm.PointerType(llvm.Int8Type(), 0)
	checkStackTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int64Type(), cStrTyp, cStrTyp}, false)
	checkCallStackTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{cStrTyp, cStrTyp}, false)

	m.printc = llvm.AddFunction(m.module, "printc", printcTyp)
	m.printi = llvm.AddFunction(m.module, "printi", printiTyp)
	m.readc = llvm.AddFunction(m.module, "readc", readcTyp)
	m.readi = llvm.AddFunction(m.module, "readi", readiTyp)
	m.flush = llvm.AddFunction(m.module, "flush", flushTyp)
	m.checkStack = llvm.AddFunction(m.module, "check_stack", checkStackTyp)
	m.checkCallStack = llvm.AddFunction(m.module, "check_call_stack", checkCallStackTyp)

	m.printc.SetLinkage(llvm.ExternalLinkage)
	m.printi.SetLinkage(llvm.ExternalLinkage)
	m.readc.SetLinkage(llvm.ExternalLinkage)
	m.readi.SetLinkage(llvm.ExternalLinkage)
	m.flush.SetLinkage(llvm.ExternalLinkage)
	m.checkStack.SetLinkage(llvm.ExternalLinkage)
	m.checkCallStack.SetLinkage(llvm.ExternalLinkage)
}

func (m *moduleBuilder) declareGlobals() {
	stackTyp := llvm.ArrayType(llvm.Int64Type(), int(m.config.MaxStackLen))
	callStackTyp := llvm.ArrayType(llvm.PointerType(llvm.Int8Type(), 0), int(m.config.MaxCallStackLen))
	heapTyp := llvm.ArrayType(llvm.Int64Type(), int(m.config.MaxHeapBound))

	m.stackLen = llvm.AddGlobal(m.module, llvm.Int64Type(), "stack_len")
	m.stack = llvm.AddGlobal(m.module, stackTyp, "stack")
	m.callStack = llvm.AddGlobal(m.module, callStackTyp, "call_stack")
	m.callStackLen = llvm.AddGlobal(m.module, llvm.Int64Type(), "call_stack_len")
	m.heap = llvm.AddGlobal(m.module, heapTyp, "heap")

	m.stack.SetInitializer(llvm.ConstNull(stackTyp))
	m.stackLen.SetInitializer(zero)
	m.callStack.SetInitializer(llvm.ConstNull(callStackTyp))
	m.callStackLen.SetInitializer(zero)
	m.heap.SetInitializer(llvm.ConstNull(heapTyp))

	for _, block := range m.program.Blocks {
		m.blockNames[block] = m.constString(block.Name())
	}
}

func (m *moduleBuilder) emitBlocks() {
	entry := m.ctx.AddBasicBlock(m.main, "")
	for _, block := range m.program.Blocks {
		m.blocks[block] = m.ctx.AddBasicBlock(m.main, block.Name())
	}

	m.b.SetInsertPoint(entry, entry.FirstInstruction())
	m.b.CreateBr(m.blocks[m.program.Entry])
	for _, block := range m.program.Blocks {
		llvmBlock := m.blocks[block]
		m.b.SetInsertPoint(llvmBlock, llvmBlock.FirstInstruction())
		stackLen := m.b.CreateLoad(m.stackLen, "stack_len")
		for _, node := range block.Nodes {
			stackLen = m.emitNode(node, block, stackLen)
		}
		m.emitTerminator(block)
	}
}

func (m *moduleBuilder) emitNode(node ir.Inst, block *ir.BasicBlock, stackLen llvm.Value) llvm.Value {
	switch inst := node.(type) {
	case *ir.BinaryExpr:
		lhs := m.lookupUse(ir.Operand(inst, 0))
		rhs := m.lookupUse(ir.Operand(inst, 1))
		var val llvm.Value
		switch inst.Op {
		case ir.Add:
			val = m.b.CreateAdd(lhs, rhs, "add")
		case ir.Sub:
			val = m.b.CreateSub(lhs, rhs, "sub")
		case ir.Mul:
			val = m.b.CreateMul(lhs, rhs, "mul")
		case ir.Div:
			val = m.b.CreateSDiv(lhs, rhs, "div")
		case ir.Mod:
			val = m.b.CreateSRem(lhs, rhs, "mod")
		case ir.Shl:
			val = m.b.CreateShl(lhs, rhs, "shl")
		case ir.LShr:
			val = m.b.CreateLShr(lhs, rhs, "lshr")
		case ir.AShr:
			val = m.b.CreateAShr(lhs, rhs, "ashr")
		case ir.And:
			val = m.b.CreateAnd(lhs, rhs, "and")
		case ir.Or:
			val = m.b.CreateOr(lhs, rhs, "or")
		case ir.Xor:
			val = m.b.CreateXor(lhs, rhs, "xor")
		default:
			panic("codegen: unrecognized binary op")
		}
		m.defs[inst] = val
	case *ir.UnaryExpr:
		switch inst.Op {
		case ir.Neg:
			val := m.lookupUse(ir.Operand(inst, 0))
			m.defs[inst] = m.b.CreateSub(zero, val, "neg")
		default:
			panic("codegen: unrecognized unary op")
		}
	case *ir.LoadStackExpr:
		addr := m.stackAddr(inst.StackPos, stackLen)
		m.defs[inst] = m.b.CreateLoad(addr, "loadstack")
	case *ir.StoreStackStmt:
		addr := m.stackAddr(inst.StackPos, stackLen)
		val := m.lookupUse(ir.Operand(inst, 0))
		m.b.CreateStore(val, addr)
	case *ir.CheckStackStmt:
		if inst.StackSize <= 0 {
			panic(fmt.Sprintf("codegen: invalid access count: %d", inst.StackSize))
		}
		n := llvm.ConstInt(llvm.Int64Type(), uint64(inst.StackSize), false)
		m.b.CreateCall(m.checkStack, []llvm.Value{n, m.blockName(block), m.instPos(inst)}, "")
	case *ir.OffsetStackStmt:
		n := llvm.ConstInt(llvm.Int64Type(), uint64(inst.Offset), false)
		stackLen = m.b.CreateAdd(stackLen, n, "offsetstack")
		m.b.CreateStore(stackLen, m.stackLen)
	case *ir.LoadHeapExpr:
		addr := m.heapAddr(ir.Operand(inst, 0))
		m.defs[inst] = m.b.CreateLoad(addr, "loadheap")
	case *ir.StoreHeapStmt:
		addr := m.heapAddr(ir.Operand(inst, 0))
		val := m.lookupUse(ir.Operand(inst, 1))
		m.b.CreateStore(val, addr)
	case *ir.PrintStmt:
		var f llvm.Value
		switch inst.Op {
		case ir.Printc:
			f = m.printc
		case ir.Printi:
			f = m.printi
		default:
			panic("codegen: unrecognized print op")
		}
		val := m.lookupUse(ir.Operand(inst, 0))
		m.b.CreateCall(f, []llvm.Value{val}, "")
	case *ir.ReadExpr:
		var f llvm.Value
		switch inst.Op {
		case ir.Readc:
			f = m.readc
		case ir.Readi:
			f = m.readi
		default:
			panic("codegen: unrecognized read op")
		}
		m.defs[inst] = m.b.CreateCall(f, []llvm.Value{}, "read")
	case *ir.FlushStmt:
		m.b.CreateCall(m.flush, []llvm.Value{}, "")
	default:
		panic("codegen: unrecognized node type")
	}
	return stackLen
}

func (m *moduleBuilder) emitTerminator(block *ir.BasicBlock) {
	switch term := block.Terminator.(type) {
	case *ir.CallTerm:
		callStackLen := m.b.CreateLoad(m.callStackLen, "call_stack_len")
		gep := m.b.CreateInBoundsGEP(m.callStack, []llvm.Value{zero, callStackLen}, "ret_addr.gep")
		callStackLen = m.b.CreateAdd(callStackLen, one, "call_stack_len")
		m.b.CreateStore(callStackLen, m.callStackLen)
		addr := llvm.BlockAddress(m.main, m.blocks[ir.Succ(term, 1)])
		m.b.CreateStore(addr, gep)
		m.b.CreateBr(m.blocks[ir.Succ(term, 0)])
	case *ir.JmpTerm:
		m.b.CreateBr(m.blocks[ir.Succ(term, 0)])
	case *ir.JmpCondTerm:
		val := m.lookupUse(ir.Operand(term, 0))
		var cond llvm.Value
		switch term.Op {
		case ir.Jz:
			cond = m.b.CreateICmp(llvm.IntEQ, val, zero, "cmp")
		case ir.Jn:
			cond = m.b.CreateICmp(llvm.IntSLT, val, zero, "cmp")
		default:
			panic("codegen: unrecognized conditional jump op")
		}
		m.b.CreateCondBr(cond, m.blocks[ir.Succ(term, 0)], m.blocks[ir.Succ(term, 1)])
	case *ir.RetTerm:
		m.b.CreateCall(m.checkCallStack, []llvm.Value{m.blockName(block), m.instPos(term)}, "")
		callStackLen := m.b.CreateLoad(m.callStackLen, "call_stack_len")
		callStackLen = m.b.CreateSub(callStackLen, one, "call_stack_len")
		m.b.CreateStore(callStackLen, m.callStackLen)
		gep := m.b.CreateInBoundsGEP(m.callStack, []llvm.Value{zero, callStackLen}, "ret_addr.gep")
		addr := m.b.CreateLoad(gep, "ret_addr")
		dests := block.Exits()
		br := m.b.CreateIndirectBr(addr, len(dests))
		for _, dest := range dests {
			if dest != nil {
				br.AddDest(m.blocks[dest])
			}
		}
	case *ir.ExitTerm:
		m.b.CreateRet(llvm.ConstInt(llvm.Int32Type(), 0, false))
	default:
		panic("codegen: unrecognized terminator type")
	}
}

func (m *moduleBuilder) lookupUse(use *ir.ValueUse) llvm.Value {
	return m.lookupValue(use.Def)
}

func (m *moduleBuilder) lookupValue(val ir.Value) llvm.Value {
	switch v := val.(type) {
	case *ir.IntConst:
		if i64, ok := bigint.ToInt64(v.Int); ok {
			return llvm.ConstInt(llvm.Int64Type(), uint64(i64), false)
		}
		panic(fmt.Sprintf("codegen: val overflows 64 bits: %v at %v", v, m.program.Position(v.Pos())))
	default:
		if ident, ok := m.defs[v]; ok {
			return ident
		}
		panic(fmt.Sprintf("codegen: def not found: %v at %v", v, m.program.Position(v.Pos())))
	}
}

func (m *moduleBuilder) stackAddr(pos int, stackLen llvm.Value) llvm.Value {
	name := fmt.Sprintf("s%d", pos)
	n := llvm.ConstInt(llvm.Int64Type(), uint64(pos), false)
	idx := m.b.CreateSub(stackLen, n, name+".idx")
	return m.b.CreateInBoundsGEP(m.stack, []llvm.Value{zero, idx}, name+".gep")
}

func (m *moduleBuilder) heapAddr(val *ir.ValueUse) llvm.Value {
	addr := m.lookupUse(val)
	return m.b.CreateInBoundsGEP(m.heap, []llvm.Value{zero, addr}, "gep")
}

func (m *moduleBuilder) constString(str string) llvm.Value {
	global := llvm.AddGlobal(m.module, llvm.ArrayType(llvm.Int8Type(), len(str)+1), "str_"+str)
	global.SetInitializer(m.ctx.ConstString(str, true))
	global.SetLinkage(llvm.PrivateLinkage)
	return global
}

func (m *moduleBuilder) blockName(block *ir.BasicBlock) llvm.Value {
	return m.b.CreateInBoundsGEP(m.blockNames[block], []llvm.Value{zero, zero}, "name")
}

func (m *moduleBuilder) instPos(inst ir.Inst) llvm.Value {
	pos := m.program.Position(inst.Pos()).String()
	return m.b.CreateInBoundsGEP(m.constString(pos), []llvm.Value{zero, zero}, "op")
}
