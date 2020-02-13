package codegen // import "github.com/andrewarchi/nebula/codegen"

import (
	"fmt"

	"github.com/andrewarchi/nebula/bigint"
	"github.com/andrewarchi/nebula/ir"
	"llvm.org/llvm/bindings/go/llvm"
)

type defs struct {
	Stack        llvm.Value
	StackLen     llvm.Value
	CallStack    llvm.Value
	CallStackLen llvm.Value
	Heap         llvm.Value

	MainFunc           llvm.Value
	PrintcFunc         llvm.Value
	PrintiFunc         llvm.Value
	ReadcFunc          llvm.Value
	ReadiFunc          llvm.Value
	FlushFunc          llvm.Value
	CheckStackFunc     llvm.Value
	CheckCallStackFunc llvm.Value

	BlockNames map[*ir.BasicBlock]llvm.Value
}

// Config stores configuration of allocation sizes for codegen.
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
func EmitLLVMModule(program *ir.Program, conf Config) llvm.Module {
	ctx := llvm.GlobalContext()
	b := ctx.NewBuilder()
	module := ctx.NewModule(program.Name)

	var d defs
	d.BlockNames = make(map[*ir.BasicBlock]llvm.Value)
	d.declareFuncs(module)
	d.declareGlobals(ctx, module, program.Blocks, conf)

	entry := ctx.AddBasicBlock(d.MainFunc, "")
	blocks := make(map[*ir.BasicBlock]llvm.BasicBlock)
	for _, block := range program.Blocks {
		blocks[block] = ctx.AddBasicBlock(d.MainFunc, block.Name())
	}

	b.SetInsertPoint(entry, entry.FirstInstruction())
	b.CreateBr(blocks[program.Entry])
	for _, block := range program.Blocks {
		llvmBlock := blocks[block]
		b.SetInsertPoint(llvmBlock, llvmBlock.FirstInstruction())
		idents := make(map[ir.Val]llvm.Value)
		stackLen := b.CreateLoad(d.StackLen, "stack_len")
		for _, node := range block.Nodes {
			d.emitNode(b, node, block, idents, stackLen)
		}
		d.updateStack(b, block, idents, stackLen)
		d.emitTerminator(b, block, idents, blocks)
	}
	return module
}

func (d *defs) declareFuncs(module llvm.Module) {
	mainTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{}, false)
	d.MainFunc = llvm.AddFunction(module, "main", mainTyp)

	printcTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int64Type()}, false)
	printiTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int64Type()}, false)
	readcTyp := llvm.FunctionType(llvm.Int64Type(), []llvm.Type{}, false)
	readiTyp := llvm.FunctionType(llvm.Int64Type(), []llvm.Type{}, false)
	flushTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{}, false)
	checkStackTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int64Type(), llvm.PointerType(llvm.Int8Type(), 0)}, false)
	checkCallStackTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.PointerType(llvm.Int8Type(), 0)}, false)

	d.PrintcFunc = llvm.AddFunction(module, "printc", printcTyp)
	d.PrintiFunc = llvm.AddFunction(module, "printi", printiTyp)
	d.ReadcFunc = llvm.AddFunction(module, "readc", readcTyp)
	d.ReadiFunc = llvm.AddFunction(module, "readi", readiTyp)
	d.FlushFunc = llvm.AddFunction(module, "flush", flushTyp)
	d.CheckStackFunc = llvm.AddFunction(module, "check_stack", checkStackTyp)
	d.CheckCallStackFunc = llvm.AddFunction(module, "check_call_stack", checkCallStackTyp)

	d.PrintcFunc.SetLinkage(llvm.ExternalLinkage)
	d.PrintiFunc.SetLinkage(llvm.ExternalLinkage)
	d.ReadcFunc.SetLinkage(llvm.ExternalLinkage)
	d.ReadiFunc.SetLinkage(llvm.ExternalLinkage)
	d.FlushFunc.SetLinkage(llvm.ExternalLinkage)
	d.CheckStackFunc.SetLinkage(llvm.ExternalLinkage)
	d.CheckCallStackFunc.SetLinkage(llvm.ExternalLinkage)
}

func (d *defs) declareGlobals(ctx llvm.Context, module llvm.Module, blocks []*ir.BasicBlock, conf Config) {
	stackTyp := llvm.ArrayType(llvm.Int64Type(), int(conf.MaxStackLen))
	callStackTyp := llvm.ArrayType(llvm.PointerType(llvm.Int8Type(), 0), int(conf.MaxCallStackLen))
	heapTyp := llvm.ArrayType(llvm.Int64Type(), int(conf.MaxHeapBound))

	d.StackLen = llvm.AddGlobal(module, llvm.Int64Type(), "stack_len")
	d.Stack = llvm.AddGlobal(module, stackTyp, "stack")
	d.CallStack = llvm.AddGlobal(module, callStackTyp, "call_stack")
	d.CallStackLen = llvm.AddGlobal(module, llvm.Int64Type(), "call_stack_len")
	d.Heap = llvm.AddGlobal(module, heapTyp, "heap")

	d.Stack.SetInitializer(llvm.ConstNull(stackTyp))
	d.StackLen.SetInitializer(zero)
	d.CallStack.SetInitializer(llvm.ConstNull(callStackTyp))
	d.CallStackLen.SetInitializer(zero)
	d.Heap.SetInitializer(llvm.ConstNull(heapTyp))

	for _, block := range blocks {
		name := block.Name()
		nameGlobal := llvm.AddGlobal(module, llvm.ArrayType(llvm.Int8Type(), len(name)+1), "name_"+name)
		nameGlobal.SetInitializer(ctx.ConstString(name, true))
		nameGlobal.SetLinkage(llvm.PrivateLinkage)
		d.BlockNames[block] = nameGlobal
	}
}

func (d *defs) emitNode(b llvm.Builder, node ir.Node, block *ir.BasicBlock, idents map[ir.Val]llvm.Value, stackLen llvm.Value) {
	switch inst := node.(type) {
	case *ir.BinaryExpr:
		lhs := lookupVal(inst.LHS, idents)
		rhs := lookupVal(inst.RHS, idents)
		var val llvm.Value
		switch inst.Op {
		case ir.Add:
			val = b.CreateAdd(lhs, rhs, "add")
		case ir.Sub:
			val = b.CreateSub(lhs, rhs, "sub")
		case ir.Mul:
			val = b.CreateMul(lhs, rhs, "mul")
		case ir.Div:
			val = b.CreateSDiv(lhs, rhs, "div")
		case ir.Mod:
			val = b.CreateSRem(lhs, rhs, "mod")
		case ir.Shl:
			val = b.CreateShl(lhs, rhs, "shl")
		case ir.LShr:
			val = b.CreateLShr(lhs, rhs, "lshr")
		case ir.AShr:
			val = b.CreateAShr(lhs, rhs, "ashr")
		case ir.And:
			val = b.CreateAnd(lhs, rhs, "and")
		case ir.Or:
			val = b.CreateOr(lhs, rhs, "or")
		case ir.Xor:
			val = b.CreateXor(lhs, rhs, "xor")
		default:
			panic("codegen: unrecognized binary op")
		}
		idents[*inst.Assign] = val
	case *ir.UnaryExpr:
		switch inst.Op {
		case ir.Neg:
			val := lookupVal(inst.Val, idents)
			idents[*inst.Assign] = b.CreateSub(zero, val, "neg")
		default:
			panic("codegen: unrecognized unary op")
		}
	case *ir.LoadStackExpr:
		name := fmt.Sprintf("s%d", inst.Pos)
		n := llvm.ConstInt(llvm.Int64Type(), uint64(inst.Pos), false)
		idx := b.CreateSub(stackLen, n, name+".idx")
		gep := b.CreateInBoundsGEP(d.Stack, []llvm.Value{zero, idx}, name+".gep")
		idents[*inst.Assign] = b.CreateLoad(gep, name)
	case *ir.LoadHeapExpr:
		addr := d.heapAddr(b, inst.Addr, idents)
		idents[*inst.Assign] = b.CreateLoad(addr, "retrieve")
	case *ir.StoreHeapStmt:
		addr := d.heapAddr(b, inst.Addr, idents)
		val := lookupVal(inst.Val, idents)
		b.CreateStore(val, addr)
	case *ir.CheckStackStmt:
		if inst.Access <= 0 {
			panic(fmt.Sprintf("codegen: invalid access count: %d", inst.Access))
		}
		n := llvm.ConstInt(llvm.Int64Type(), uint64(inst.Access), false)
		b.CreateCall(d.CheckStackFunc, []llvm.Value{n, d.blockName(b, block)}, "")
	case *ir.PrintStmt:
		var f llvm.Value
		switch inst.Op {
		case ir.Printc:
			f = d.PrintcFunc
		case ir.Printi:
			f = d.PrintiFunc
		default:
			panic("codegen: unrecognized print op")
		}
		val := lookupVal(inst.Val, idents)
		b.CreateCall(f, []llvm.Value{val}, "")
	case *ir.ReadExpr:
		var f llvm.Value
		switch inst.Op {
		case ir.Readc:
			f = d.ReadcFunc
		case ir.Readi:
			f = d.ReadiFunc
		default:
			panic("codegen: unrecognized read op")
		}
		idents[*inst.Assign] = b.CreateCall(f, []llvm.Value{}, "read")
	case *ir.FlushStmt:
		b.CreateCall(d.FlushFunc, []llvm.Value{}, "")
	default:
		panic("codegen: unrecognized node type")
	}
}

func (d *defs) updateStack(b llvm.Builder, block *ir.BasicBlock, idents map[ir.Val]llvm.Value, stackLen llvm.Value) {
	if pop := block.Stack.Pops; pop > 0 {
		n := llvm.ConstInt(llvm.Int64Type(), uint64(pop), false)
		stackLen = b.CreateSub(stackLen, n, "stack_len_pop")
	}
	for i, val := range block.Stack.Vals {
		v := lookupVal(val, idents)
		name := fmt.Sprintf("s%d", i)
		n := llvm.ConstInt(llvm.Int64Type(), uint64(i), false)
		idx := b.CreateAdd(stackLen, n, name+".idx")
		gep := b.CreateInBoundsGEP(d.Stack, []llvm.Value{zero, idx}, name+".gep")
		b.CreateStore(v, gep)
	}
	if push := len(block.Stack.Vals); push > 0 {
		n := llvm.ConstInt(llvm.Int64Type(), uint64(push), false)
		stackLen = b.CreateAdd(stackLen, n, "stack_len_push")
	}
	b.CreateStore(stackLen, d.StackLen)
}

func (d *defs) emitTerminator(b llvm.Builder, block *ir.BasicBlock, idents map[ir.Val]llvm.Value, blocks map[*ir.BasicBlock]llvm.BasicBlock) {
	switch term := block.Terminator.(type) {
	case *ir.CallStmt:
		callStackLen := b.CreateLoad(d.CallStackLen, "call_stack_len")
		gep := b.CreateInBoundsGEP(d.CallStack, []llvm.Value{zero, callStackLen}, "ret_addr.gep")
		callStackLen = b.CreateAdd(callStackLen, one, "call_stack_len")
		b.CreateStore(callStackLen, d.CallStackLen)
		addr := llvm.BlockAddress(d.MainFunc, blocks[block.Next])
		b.CreateStore(addr, gep)
		b.CreateBr(blocks[term.Dest])
	case *ir.JmpStmt:
		b.CreateBr(blocks[term.Dest])
	case *ir.JmpCondStmt:
		val := lookupVal(term.Cond, idents)
		var cond llvm.Value
		switch term.Op {
		case ir.Jz:
			cond = b.CreateICmp(llvm.IntEQ, val, zero, "cmp")
		case ir.Jn:
			cond = b.CreateICmp(llvm.IntSLT, val, zero, "cmp")
		default:
			panic("codegen: unrecognized conditional jump op")
		}
		b.CreateCondBr(cond, blocks[term.Then], blocks[term.Else])
	case *ir.RetStmt:
		b.CreateCall(d.CheckCallStackFunc, []llvm.Value{d.blockName(b, block)}, "")
		callStackLen := b.CreateLoad(d.CallStackLen, "call_stack_len")
		callStackLen = b.CreateSub(callStackLen, one, "call_stack_len")
		b.CreateStore(callStackLen, d.CallStackLen)
		gep := b.CreateInBoundsGEP(d.CallStack, []llvm.Value{zero, callStackLen}, "ret_addr.gep")
		addr := b.CreateLoad(gep, "ret_addr")
		dests := block.Exits()
		br := b.CreateIndirectBr(addr, len(dests))
		for _, dest := range dests {
			br.AddDest(blocks[dest])
		}
	case *ir.ExitStmt:
		b.CreateRetVoid()
	default:
		panic("codegen: unrecognized terminator type")
	}
}

func (d *defs) heapAddr(b llvm.Builder, val *ir.Val, idents map[ir.Val]llvm.Value) llvm.Value {
	addr := lookupVal(val, idents)
	return b.CreateInBoundsGEP(d.Heap, []llvm.Value{zero, addr}, "gep")
}

func (d *defs) blockName(b llvm.Builder, block *ir.BasicBlock) llvm.Value {
	return b.CreateInBoundsGEP(d.BlockNames[block], []llvm.Value{zero, zero}, "name")
}

func lookupVal(val *ir.Val, idents map[ir.Val]llvm.Value) llvm.Value {
	switch v := (*val).(type) {
	case *ir.SSAVal:
		if v, ok := idents[v]; ok {
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
