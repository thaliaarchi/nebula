package codegen

import (
	"fmt"

	"github.com/andrewarchi/nebula/bigint"
	"github.com/andrewarchi/nebula/ir"
	"github.com/andrewarchi/nebula/token"
	"llvm.org/llvm/bindings/go/llvm"
)

type builder struct {
	Program      *ir.Program
	Ctx          llvm.Context
	Builder      llvm.Builder
	Mod          llvm.Module
	Main         llvm.Value
	Blocks       map[*ir.BasicBlock]llvm.BasicBlock
	Stack        llvm.Value
	StackLen     llvm.Value
	CallStack    llvm.Value
	CallStackLen llvm.Value
	Heap         llvm.Value
	PrintcFunc   llvm.Value
	PrintiFunc   llvm.Value
	ReadcFunc    llvm.Value
	ReadiFunc    llvm.Value
	FlushFunc    llvm.Value
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
	b := builder{
		Program: program,
		Ctx:     ctx,
		Builder: ctx.NewBuilder(),
		Mod:     ctx.NewModule(program.Name),
	}

	b.declareFuncs()
	b.Blocks = make(map[*ir.BasicBlock]llvm.BasicBlock)
	entry := b.Ctx.AddBasicBlock(b.Main, "entry")
	for _, block := range program.Blocks {
		b.Blocks[block] = b.Ctx.AddBasicBlock(b.Main, block.Name())
	}

	b.emitEntry(entry)
	for _, block := range program.Blocks {
		b.emitBlock(block, b.Blocks[block])
	}
	return b.Mod
}

func (b *builder) declareFuncs() {
	mainTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{}, false)
	printcTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int64Type()}, false)
	printiTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int64Type()}, false)
	readcTyp := llvm.FunctionType(llvm.Int64Type(), []llvm.Type{}, false)
	readiTyp := llvm.FunctionType(llvm.Int64Type(), []llvm.Type{}, false)
	flushTyp := llvm.FunctionType(llvm.VoidType(), []llvm.Type{}, false)
	b.Main = llvm.AddFunction(b.Mod, "main", mainTyp)
	b.PrintcFunc = llvm.AddFunction(b.Mod, "printc", printcTyp)
	b.PrintiFunc = llvm.AddFunction(b.Mod, "printi", printiTyp)
	b.ReadcFunc = llvm.AddFunction(b.Mod, "readc", readcTyp)
	b.ReadiFunc = llvm.AddFunction(b.Mod, "readi", readiTyp)
	b.FlushFunc = llvm.AddFunction(b.Mod, "flush", flushTyp)
	b.PrintcFunc.SetLinkage(llvm.ExternalLinkage)
	b.PrintiFunc.SetLinkage(llvm.ExternalLinkage)
	b.ReadcFunc.SetLinkage(llvm.ExternalLinkage)
	b.ReadiFunc.SetLinkage(llvm.ExternalLinkage)
	b.FlushFunc.SetLinkage(llvm.ExternalLinkage)
}

func (b *builder) emitEntry(entry llvm.BasicBlock) {
	b.Builder.SetInsertPoint(entry, entry.FirstInstruction())
	b.Stack = b.Builder.CreateAlloca(llvm.ArrayType(llvm.Int64Type(), maxStackSize), "stack")
	b.StackLen = b.Builder.CreateAlloca(llvm.Int64Type(), "stack_len")
	b.CallStack = b.Builder.CreateAlloca(llvm.ArrayType(llvm.PointerType(llvm.Int8Type(), 0), maxCallStackSize), "call_stack")
	b.CallStackLen = b.Builder.CreateAlloca(llvm.Int64Type(), "call_stack_len")
	b.Heap = b.Builder.CreateAlloca(llvm.ArrayType(llvm.Int64Type(), heapSize), "heap")
	b.Builder.CreateStore(zero, b.StackLen)
	b.Builder.CreateStore(zero, b.CallStackLen)
	b.Builder.CreateBr(b.Blocks[b.Program.Entry])
}

func (b *builder) emitBlock(block *ir.BasicBlock, llvmBlock llvm.BasicBlock) {
	b.Builder.SetInsertPoint(llvmBlock, llvmBlock.FirstInstruction())
	idents := make(map[ir.Val]llvm.Value)
	if block.Stack.Access > 0 {
		// TODO check stack underflow
	}
	stackLen := b.Builder.CreateLoad(b.StackLen, "stack_len")

	for _, val := range block.Stack.Under {
		if val != nil {
			switch v := (*val).(type) {
			case *ir.StackVal:
				if v.Val < 0 {
					name := fmt.Sprintf("s%d", v.Val)
					n := llvm.ConstInt(llvm.Int64Type(), uint64(-v.Val), false)
					idx := b.Builder.CreateSub(stackLen, n, name+".idx")
					gep := b.Builder.CreateInBoundsGEP(b.Stack, []llvm.Value{zero, idx}, name+".gep")
					idents[v] = b.Builder.CreateLoad(gep, name)
				} else {
					panic(fmt.Sprintf("codegen: non-negative stack vals not currently supported: %v", v)) // TODO
				}
			case *ir.ConstVal:
				if i64, ok := bigint.ToInt64(v.Val); ok {
					idents[v] = llvm.ConstInt(llvm.Int64Type(), uint64(i64), false)
				} else {
					panic(fmt.Sprintf("codegen: val overflows 64 bits: %v", v))
				}
			}
		}
	}

	for _, node := range block.Nodes {
		switch inst := node.(type) {
		case *ir.AssignStmt:
			var val llvm.Value
			switch expr := inst.Expr.(type) {
			case *ir.ArithExpr:
				lhs := lookupVal(*expr.LHS, idents)
				rhs := lookupVal(*expr.RHS, idents)
				switch expr.Op {
				case token.Add:
					val = b.Builder.CreateAdd(lhs, rhs, "add")
				case token.Sub:
					val = b.Builder.CreateSub(lhs, rhs, "sub")
				case token.Mul:
					val = b.Builder.CreateMul(lhs, rhs, "mul")
				case token.Div:
					val = b.Builder.CreateSDiv(lhs, rhs, "div")
				case token.Mod:
					val = b.Builder.CreateSRem(lhs, rhs, "mod")
				}
			case *ir.RetrieveExpr:
				val = b.Builder.CreateLoad(b.heapAddr(*expr.Addr, idents), "retrieve")
			case *ir.ReadExpr:
				var f llvm.Value
				switch expr.Op {
				case token.Readc:
					f = b.ReadcFunc
				case token.Readi:
					f = b.ReadiFunc
				}
				val = b.Builder.CreateCall(f, []llvm.Value{}, "")
			}
			idents[*inst.Assign] = val
		case *ir.StoreExpr:
			b.Builder.CreateStore(lookupVal(*inst.Val, idents), b.heapAddr(*inst.Addr, idents))
		case *ir.PrintStmt:
			var f llvm.Value
			switch inst.Op {
			case token.Printc:
				f = b.PrintcFunc
			case token.Printi:
				f = b.PrintiFunc
			}
			val := lookupVal(*inst.Val, idents)
			b.Builder.CreateCall(f, []llvm.Value{val}, "")
		}
	}

	if pop := block.Stack.Pops; pop > 0 {
		n := llvm.ConstInt(llvm.Int64Type(), uint64(pop), false)
		stackLen = b.Builder.CreateSub(stackLen, n, "stack_len_pop")
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
			if i64, ok := bigint.ToInt64(v.Val); ok {
				s = llvm.ConstInt(llvm.Int64Type(), uint64(i64), false)
			} else {
				panic(fmt.Sprintf("codegen: val overflows 64 bits: %v", v))
			}
		default:
			panic(fmt.Sprintf("codegen: unsupported type %T", v))
		}
		name := fmt.Sprintf("s%d", i)
		n := llvm.ConstInt(llvm.Int64Type(), uint64(i), false)
		idx := b.Builder.CreateAdd(stackLen, n, name+"idx")
		gep := b.Builder.CreateInBoundsGEP(b.Stack, []llvm.Value{zero, idx}, name+".gep")
		b.Builder.CreateStore(s, gep)
	}
	if push := len(block.Stack.Vals); push > 0 {
		n := llvm.ConstInt(llvm.Int64Type(), uint64(push), false)
		stackLen = b.Builder.CreateAdd(stackLen, n, "stack_len_push")
	}
	b.Builder.CreateStore(stackLen, b.StackLen)
	b.emitTerminator(block, idents)
}

func (b *builder) emitTerminator(block *ir.BasicBlock, idents map[ir.Val]llvm.Value) {
	switch term := block.Terminator.(type) {
	case *ir.CallStmt:
		callStackLen := b.Builder.CreateLoad(b.CallStackLen, "call_stack_len")
		gep := b.Builder.CreateInBoundsGEP(b.CallStack, []llvm.Value{zero, callStackLen}, "ret_addr.gep")
		callStackLen = b.Builder.CreateAdd(callStackLen, one, "call_stack_len")
		b.Builder.CreateStore(callStackLen, b.CallStackLen)
		addr := llvm.BlockAddress(b.Main, b.Blocks[block.Next])
		b.Builder.CreateStore(addr, gep)
		b.Builder.CreateBr(b.Blocks[term.Callee])
	case *ir.JmpStmt:
		b.Builder.CreateBr(b.Blocks[term.Block])
	case *ir.JmpCondStmt:
		val := idents[*term.Cond]
		var cond llvm.Value
		switch term.Op {
		case token.Jz:
			cond = b.Builder.CreateICmp(llvm.IntEQ, val, zero, "cmp")
		case token.Jn:
			cond = b.Builder.CreateICmp(llvm.IntSLT, val, zero, "cmp")
		}
		b.Builder.CreateCondBr(cond, b.Blocks[term.ThenBlock], b.Blocks[term.ElseBlock])
	case *ir.RetStmt:
		callStackLen := b.Builder.CreateLoad(b.CallStackLen, "call_stack_len")
		callStackLen = b.Builder.CreateSub(callStackLen, one, "call_stack_len")
		// TODO check call stack underflow
		b.Builder.CreateStore(callStackLen, b.CallStackLen)
		gep := b.Builder.CreateInBoundsGEP(b.CallStack, []llvm.Value{zero, callStackLen}, "ret_addr.gep")
		addr := b.Builder.CreateLoad(gep, "ret_addr")
		dests := block.Exits()
		br := b.Builder.CreateIndirectBr(addr, len(dests))
		for _, dest := range dests {
			br.AddDest(b.Blocks[dest])
		}
	case *ir.EndStmt:
		b.Builder.CreateRetVoid()
	}
}

func (b *builder) heapAddr(val ir.Val, idents map[ir.Val]llvm.Value) llvm.Value {
	addr := lookupVal(val, idents)
	return b.Builder.CreateInBoundsGEP(b.Heap, []llvm.Value{zero, addr}, "gep")
}

func lookupVal(val ir.Val, idents map[ir.Val]llvm.Value) llvm.Value {
	switch v := val.(type) {
	case *ir.StackVal:
		if v, ok := idents[val]; ok {
			return v
		}
		panic(fmt.Sprintf("codegen: val not found: %v", val))
	case *ir.ConstVal:
		if i64, ok := bigint.ToInt64(v.Val); ok {
			return llvm.ConstInt(llvm.Int64Type(), uint64(i64), false)
		}
		panic(fmt.Sprintf("codegen: val overflows 64 bits: %v", val))
	default:
		panic(fmt.Sprintf("codegen: val type not supported: %v", val))
	}
}
