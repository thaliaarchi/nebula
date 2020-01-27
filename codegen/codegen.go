package codegen

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
}

const (
	maxStackSize     = 1024
	maxCallStackSize = 256
	heapSize         = 4096
)

var (
	zero = llvm.ConstInt(llvm.Int64Type(), 0, false)
	one  = llvm.ConstInt(llvm.Int64Type(), 1, false)
)

func EmitLLVMIR(program *ir.Program) llvm.Module {
	ctx := llvm.GlobalContext()
	b := ctx.NewBuilder()
	module := ctx.NewModule(program.Name)
	var d defs
	d.declareFuncs(module)
	d.declareGlobals(module)

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
		idents, stackLen := d.loadStack(b, block)
		for _, node := range block.Nodes {
			d.emitNode(b, node, idents)
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
	checkStackTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int64Type()}, false)
	checkCallStackTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{}, false)
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

func (d *defs) declareGlobals(module llvm.Module) {
	stackTyp := llvm.ArrayType(llvm.Int64Type(), maxStackSize)
	callStackTyp := llvm.ArrayType(llvm.PointerType(llvm.Int8Type(), 0), maxCallStackSize)
	heapTyp := llvm.ArrayType(llvm.Int64Type(), heapSize)
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
}

func (d *defs) loadStack(b llvm.Builder, block *ir.BasicBlock) (map[ir.Val]llvm.Value, llvm.Value) {
	idents := make(map[ir.Val]llvm.Value)
	if block.Stack.Access > 0 {
		n := llvm.ConstInt(llvm.Int64Type(), uint64(block.Stack.Access), false)
		b.CreateCall(d.CheckStackFunc, []llvm.Value{n}, "")
	}
	stackLen := b.CreateLoad(d.StackLen, "stack_len")

	for _, val := range block.Stack.Under {
		if val != nil {
			switch v := (*val).(type) {
			case *ir.StackVal:
				if v.ID < 0 {
					name := fmt.Sprintf("s%d", v.ID)
					n := llvm.ConstInt(llvm.Int64Type(), uint64(-v.ID), false)
					idx := b.CreateSub(stackLen, n, name+".idx")
					gep := b.CreateInBoundsGEP(d.Stack, []llvm.Value{zero, idx}, name+".gep")
					idents[v] = b.CreateLoad(gep, name)
				} else {
					panic(fmt.Sprintf("codegen: non-negative stack vals not currently supported: %v", v)) // TODO
				}
			case *ir.ConstVal:
				if i64, ok := bigint.ToInt64(v.Int); ok {
					idents[v] = llvm.ConstInt(llvm.Int64Type(), uint64(i64), false)
				} else {
					panic(fmt.Sprintf("codegen: val overflows 64 bits: %v", v))
				}
			}
		}
	}
	return idents, stackLen
}

func (d *defs) emitNode(b llvm.Builder, node ir.Node, idents map[ir.Val]llvm.Value) {
	switch inst := node.(type) {
	case *ir.BinaryExpr:
		lhs := lookupVal(*inst.LHS, idents)
		rhs := lookupVal(*inst.RHS, idents)
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
		}
		idents[*inst.Assign] = val
	case *ir.UnaryExpr:
		switch inst.Op {
		case ir.Neg:
			val := lookupVal(*inst.Val, idents)
			idents[*inst.Assign] = b.CreateSub(zero, val, "neg")
		}
	case *ir.LoadExpr:
		addr := d.heapAddr(b, *inst.Addr, idents)
		idents[*inst.Assign] = b.CreateLoad(addr, "retrieve")
	case *ir.StoreStmt:
		addr := d.heapAddr(b, *inst.Addr, idents)
		val := lookupVal(*inst.Val, idents)
		b.CreateStore(val, addr)
	case *ir.PrintStmt:
		var f llvm.Value
		switch inst.Op {
		case ir.Printc:
			f = d.PrintcFunc
		case ir.Printi:
			f = d.PrintiFunc
		}
		val := lookupVal(*inst.Val, idents)
		b.CreateCall(f, []llvm.Value{val}, "")
	case *ir.ReadExpr:
		var f llvm.Value
		switch inst.Op {
		case ir.Readc:
			f = d.ReadcFunc
		case ir.Readi:
			f = d.ReadiFunc
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
		var s llvm.Value
		switch v := (*val).(type) {
		case *ir.StackVal:
			if ident, ok := idents[v]; ok {
				s = ident
			} else {
				panic(fmt.Sprintf("codegen: val not in scope of %s: %v", block.Name(), *val))
			}
		case *ir.ConstVal:
			if i64, ok := bigint.ToInt64(v.Int); ok {
				s = llvm.ConstInt(llvm.Int64Type(), uint64(i64), false)
			} else {
				panic(fmt.Sprintf("codegen: val overflows 64 bits: %v", v))
			}
		default:
			panic("codegen: unrecognized val type")
		}
		name := fmt.Sprintf("s%d", i)
		n := llvm.ConstInt(llvm.Int64Type(), uint64(i), false)
		idx := b.CreateAdd(stackLen, n, name+"idx")
		gep := b.CreateInBoundsGEP(d.Stack, []llvm.Value{zero, idx}, name+".gep")
		b.CreateStore(s, gep)
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
		val := idents[*term.Cond]
		var cond llvm.Value
		switch term.Op {
		case ir.Jz:
			cond = b.CreateICmp(llvm.IntEQ, val, zero, "cmp")
		case ir.Jn:
			cond = b.CreateICmp(llvm.IntSLT, val, zero, "cmp")
		}
		b.CreateCondBr(cond, blocks[term.Then], blocks[term.Else])
	case *ir.RetStmt:
		callStackLen := b.CreateLoad(d.CallStackLen, "call_stack_len")
		callStackLen = b.CreateSub(callStackLen, one, "call_stack_len")
		b.CreateCall(d.CheckCallStackFunc, []llvm.Value{}, "")
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
		panic("codegen: unrecognized branch type")
	}
}

func (d *defs) heapAddr(b llvm.Builder, val ir.Val, idents map[ir.Val]llvm.Value) llvm.Value {
	addr := lookupVal(val, idents)
	return b.CreateInBoundsGEP(d.Heap, []llvm.Value{zero, addr}, "gep")
}

func lookupVal(val ir.Val, idents map[ir.Val]llvm.Value) llvm.Value {
	switch v := val.(type) {
	case *ir.StackVal:
		if v, ok := idents[val]; ok {
			return v
		}
		panic(fmt.Sprintf("codegen: val not found: %v", val))
	case *ir.ConstVal:
		if i64, ok := bigint.ToInt64(v.Int); ok {
			return llvm.ConstInt(llvm.Int64Type(), uint64(i64), false)
		}
		panic(fmt.Sprintf("codegen: val overflows 64 bits: %v", val))
	default:
		panic("codegen: unrecognized val type")
	}
}
