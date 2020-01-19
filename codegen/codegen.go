package codegen

import (
	"fmt"

	"github.com/andrewarchi/nebula/bigint"
	"github.com/andrewarchi/nebula/ir"
	"github.com/andrewarchi/nebula/token"
	"llvm.org/llvm/bindings/go/llvm"
)

type builder struct {
	Program  *ir.Program
	Ctx      llvm.Context
	Builder  llvm.Builder
	Mod      llvm.Module
	Main     llvm.Value
	Entry    llvm.BasicBlock
	Blocks   map[*ir.BasicBlock]llvm.BasicBlock
	Stack    llvm.Value
	StackLen llvm.Value
}

const maxStackSize = 1024

var zero = llvm.ConstInt(llvm.Int64Type(), 0, false)

func EmitLLVMIR(program *ir.Program) {
	ctx := llvm.GlobalContext()
	b := builder{
		Program: program,
		Ctx:     ctx,
		Builder: ctx.NewBuilder(),
		Mod:     ctx.NewModule(program.Name),
	}

	mainType := llvm.FunctionType(llvm.Int64Type(), []llvm.Type{}, false)
	b.Main = llvm.AddFunction(b.Mod, "main", mainType)
	b.Entry = llvm.AddBasicBlock(b.Main, "entry")
	b.Builder.SetInsertPoint(b.Entry, b.Entry.FirstInstruction())

	b.Stack = b.Builder.CreateAlloca(llvm.ArrayType(llvm.Int64Type(), maxStackSize), "stack") // should be global
	b.StackLen = b.Builder.CreateAlloca(llvm.Int64Type(), "stack_len")                        // should be global

	xIdx := llvm.ConstInt(llvm.Int64Type(), 5, false)
	yIdx := llvm.ConstInt(llvm.Int64Type(), 4, false)
	xGep := b.Builder.CreateInBoundsGEP(b.Stack, []llvm.Value{zero, xIdx}, "x.gep")
	yGep := b.Builder.CreateInBoundsGEP(b.Stack, []llvm.Value{zero, yIdx}, "y.gep")
	b.Builder.CreateStore(llvm.ConstInt(llvm.Int64Type(), 16, false), xGep)
	b.Builder.CreateStore(llvm.ConstInt(llvm.Int64Type(), 42, false), yGep)
	x := b.Builder.CreateLoad(xGep, "x")
	y := b.Builder.CreateLoad(yGep, "y")
	xy := b.Builder.CreateAdd(x, y, "xy")
	b.Builder.CreateRet(xy)

	for _, block := range program.Blocks {
		b.emitBlock(block)
	}

	if ok := llvm.VerifyModule(b.Mod, llvm.ReturnStatusAction); ok != nil {
		fmt.Println(ok.Error())
	}
	b.Mod.Dump()

	// engine, err := llvm.NewExecutionEngine(b.Mod)
	// if err != nil {
	// 	fmt.Println(err.Error())
	// }

	// funcResult := engine.RunFunction(b.Main, []llvm.GenericValue{})
	// fmt.Printf("%d\n", funcResult.Int(false))
}

func (b *builder) emitBlock(block *ir.BasicBlock) {
	lb := b.Ctx.AddBasicBlock(b.Main, block.Name())
	b.Builder.SetInsertPoint(lb, lb.FirstInstruction())
	idents := make(map[*ir.Val]llvm.Value)
	if block.Stack.Access > 0 {
		b.callStackCheck(block.Stack.Access)
	}
	stackLen := b.Builder.CreateLoad(b.StackLen, "stack_len")
	for _, val := range block.Stack.Under {
		if val != nil {
			if v, ok := (*val).(*ir.StackVal); !ok || v.Val < 0 {
				name := fmt.Sprintf("s%d", v.Val)
				n := llvm.ConstInt(llvm.Int64Type(), uint64(-v.Val), false)
				idx := b.Builder.CreateSub(stackLen, n, name+".idx")
				gep := b.Builder.CreateInBoundsGEP(b.Stack, []llvm.Value{zero, idx}, name+".gep")
				idents[val] = b.Builder.CreateLoad(gep, name)
			}
		}
	}
	for _, node := range block.Nodes {
		switch n := node.(type) {
		case *ir.AssignStmt:
			var val llvm.Value
			switch expr := n.Expr.(type) {
			case *ir.ArithExpr:
				lhs := valLookup(idents, expr.LHS)
				rhs := valLookup(idents, expr.RHS)
				switch expr.Op {
				case token.Add:
					val = b.Builder.CreateAdd(lhs, rhs, "")
				case token.Sub:
					val = b.Builder.CreateSub(lhs, rhs, "")
				case token.Mul:
					val = b.Builder.CreateMul(lhs, rhs, "")
				case token.Div:
					val = b.Builder.CreateSDiv(lhs, rhs, "")
				case token.Mod:
					val = b.Builder.CreateSRem(lhs, rhs, "")
				}
			case *ir.RetrieveExpr:
				val = b.Builder.CreateAlloca(llvm.Int64Type(), "retrieve") // TODO
			case *ir.ReadExpr:
				val = b.Builder.CreateAlloca(llvm.Int64Type(), "read") // TODO
			}
			idents[n.Assign] = val
		case *ir.StoreExpr:
			b.Builder.CreateAlloca(llvm.Int64Type(), "store") // TODO
		case *ir.PrintStmt:
			b.Builder.CreateAlloca(llvm.Int64Type(), "print") // TODO
		}
	}
}

func (b *builder) callStackCheck(access int) {
	// accessConst := llvm.ConstInt(llvm.Int64Type(), uint64(access), false)
	// b.StackLen
}

func valLookup(idents map[*ir.Val]llvm.Value, val *ir.Val) llvm.Value {
	switch v := (*val).(type) {
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
