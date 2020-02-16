package codegen // import "github.com/andrewarchi/nebula/codegen"

import (
	"fmt"

	"github.com/andrewarchi/nebula/bigint"
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
	idents     map[ir.Val]llvm.Value

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
		idents:     make(map[ir.Val]llvm.Value),
	}
	m.declareFuncs()
	m.declareGlobals()
	m.emitBlocks()
	return m.module
}

func (m *moduleBuilder) declareFuncs() {
	mainTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{}, false)
	m.main = llvm.AddFunction(m.module, "main", mainTyp)

	printcTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int64Type()}, false)
	printiTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int64Type()}, false)
	readcTyp := llvm.FunctionType(llvm.Int64Type(), []llvm.Type{}, false)
	readiTyp := llvm.FunctionType(llvm.Int64Type(), []llvm.Type{}, false)
	flushTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{}, false)
	checkStackTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int64Type(), llvm.PointerType(llvm.Int8Type(), 0)}, false)
	checkCallStackTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.PointerType(llvm.Int8Type(), 0)}, false)

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
		name := block.Name()
		nameGlobal := llvm.AddGlobal(m.module, llvm.ArrayType(llvm.Int8Type(), len(name)+1), "name_"+name)
		nameGlobal.SetInitializer(m.ctx.ConstString(name, true))
		nameGlobal.SetLinkage(llvm.PrivateLinkage)
		m.blockNames[block] = nameGlobal
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
			m.emitNode(node, block, stackLen)
		}
		m.updateStack(block, stackLen)
		m.emitTerminator(block)
	}
}

func (m *moduleBuilder) emitNode(node ir.Node, block *ir.BasicBlock, stackLen llvm.Value) {
	switch inst := node.(type) {
	case *ir.BinaryExpr:
		lhs := m.lookupVal(inst.LHS)
		rhs := m.lookupVal(inst.RHS)
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
		m.idents[*inst.Assign] = val
	case *ir.UnaryExpr:
		switch inst.Op {
		case ir.Neg:
			val := m.lookupVal(inst.Val)
			m.idents[*inst.Assign] = m.b.CreateSub(zero, val, "neg")
		default:
			panic("codegen: unrecognized unary op")
		}
	case *ir.LoadStackExpr:
		name := fmt.Sprintf("s%d", inst.Pos)
		n := llvm.ConstInt(llvm.Int64Type(), uint64(inst.Pos), false)
		idx := m.b.CreateSub(stackLen, n, name+".idx")
		gep := m.b.CreateInBoundsGEP(m.stack, []llvm.Value{zero, idx}, name+".gep")
		m.idents[*inst.Assign] = m.b.CreateLoad(gep, name)
	case *ir.LoadHeapExpr:
		addr := m.heapAddr(inst.Addr)
		m.idents[*inst.Assign] = m.b.CreateLoad(addr, "retrieve")
	case *ir.StoreHeapStmt:
		addr := m.heapAddr(inst.Addr)
		val := m.lookupVal(inst.Val)
		m.b.CreateStore(val, addr)
	case *ir.CheckStackStmt:
		if inst.Access <= 0 {
			panic(fmt.Sprintf("codegen: invalid access count: %d", inst.Access))
		}
		n := llvm.ConstInt(llvm.Int64Type(), uint64(inst.Access), false)
		m.b.CreateCall(m.checkStack, []llvm.Value{n, m.blockName(block)}, "")
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
		val := m.lookupVal(inst.Val)
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
		m.idents[*inst.Assign] = m.b.CreateCall(f, []llvm.Value{}, "read")
	case *ir.FlushStmt:
		m.b.CreateCall(m.flush, []llvm.Value{}, "")
	default:
		panic("codegen: unrecognized node type")
	}
}

func (m *moduleBuilder) updateStack(block *ir.BasicBlock, stackLen llvm.Value) {
	if pop := block.Stack.Pops; pop > 0 {
		n := llvm.ConstInt(llvm.Int64Type(), uint64(pop), false)
		stackLen = m.b.CreateSub(stackLen, n, "stack_len_pop")
	}
	for i, val := range block.Stack.Vals {
		v := m.lookupVal(val)
		name := fmt.Sprintf("s%d", i)
		n := llvm.ConstInt(llvm.Int64Type(), uint64(i), false)
		idx := m.b.CreateAdd(stackLen, n, name+".idx")
		gep := m.b.CreateInBoundsGEP(m.stack, []llvm.Value{zero, idx}, name+".gep")
		m.b.CreateStore(v, gep)
	}
	if push := len(block.Stack.Vals); push > 0 {
		n := llvm.ConstInt(llvm.Int64Type(), uint64(push), false)
		stackLen = m.b.CreateAdd(stackLen, n, "stack_len_push")
	}
	m.b.CreateStore(stackLen, m.stackLen)
}

func (m *moduleBuilder) emitTerminator(block *ir.BasicBlock) {
	switch term := block.Terminator.(type) {
	case *ir.CallStmt:
		callStackLen := m.b.CreateLoad(m.callStackLen, "call_stack_len")
		gep := m.b.CreateInBoundsGEP(m.callStack, []llvm.Value{zero, callStackLen}, "ret_addr.gep")
		callStackLen = m.b.CreateAdd(callStackLen, one, "call_stack_len")
		m.b.CreateStore(callStackLen, m.callStackLen)
		addr := llvm.BlockAddress(m.main, m.blocks[block.Next])
		m.b.CreateStore(addr, gep)
		m.b.CreateBr(m.blocks[term.Dest])
	case *ir.JmpStmt:
		m.b.CreateBr(m.blocks[term.Dest])
	case *ir.JmpCondStmt:
		val := m.lookupVal(term.Cond)
		var cond llvm.Value
		switch term.Op {
		case ir.Jz:
			cond = m.b.CreateICmp(llvm.IntEQ, val, zero, "cmp")
		case ir.Jn:
			cond = m.b.CreateICmp(llvm.IntSLT, val, zero, "cmp")
		default:
			panic("codegen: unrecognized conditional jump op")
		}
		m.b.CreateCondBr(cond, m.blocks[term.Then], m.blocks[term.Else])
	case *ir.RetStmt:
		m.b.CreateCall(m.checkCallStack, []llvm.Value{m.blockName(block)}, "")
		callStackLen := m.b.CreateLoad(m.callStackLen, "call_stack_len")
		callStackLen = m.b.CreateSub(callStackLen, one, "call_stack_len")
		m.b.CreateStore(callStackLen, m.callStackLen)
		gep := m.b.CreateInBoundsGEP(m.callStack, []llvm.Value{zero, callStackLen}, "ret_addr.gep")
		addr := m.b.CreateLoad(gep, "ret_addr")
		dests := block.Exits()
		br := m.b.CreateIndirectBr(addr, len(dests))
		for _, dest := range dests {
			br.AddDest(m.blocks[dest])
		}
	case *ir.ExitStmt:
		m.b.CreateRetVoid()
	default:
		panic("codegen: unrecognized terminator type")
	}
}

func (m *moduleBuilder) lookupVal(val *ir.Val) llvm.Value {
	switch v := (*val).(type) {
	case *ir.SSAVal:
		if v, ok := m.idents[v]; ok {
			return v
		}
		panic(fmt.Sprintf("codegen: val not found: %v", v))
	case *ir.ConstVal:
		if i64, ok := bigint.ToInt64(v.Int); ok {
			return llvm.ConstInt(llvm.Int64Type(), uint64(i64), false)
		}
		panic(fmt.Sprintf("codegen: val overflows 64 bits: %v", v))
	default:
		panic("codegen: unrecognized val type")
	}
}

func (m *moduleBuilder) heapAddr(val *ir.Val) llvm.Value {
	addr := m.lookupVal(val)
	return m.b.CreateInBoundsGEP(m.heap, []llvm.Value{zero, addr}, "gep")
}

func (m *moduleBuilder) blockName(block *ir.BasicBlock) llvm.Value {
	return m.b.CreateInBoundsGEP(m.blockNames[block], []llvm.Value{zero, zero}, "name")
}
