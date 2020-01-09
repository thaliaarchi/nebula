package codegen

import (
	"fmt"

	"github.com/andrewarchi/nebula/ir"
	"github.com/andrewarchi/nebula/token"
	"llvm.org/llvm/bindings/go/llvm"
)

type builder struct {
	Program  *ir.Program
	Builder  llvm.Builder
	Ctx      llvm.Context
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
	var b builder
	b.Program = program
	b.Builder = llvm.NewBuilder()
	b.Mod = llvm.NewModule(program.Name)

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

	op := token.Add
	var result llvm.Value
	switch op {
	case token.Add:
		result = b.Builder.CreateAdd(x, y, "xy.add")
	case token.Sub:
		result = b.Builder.CreateSub(x, y, "xy.sub")
	case token.Mul:
		result = b.Builder.CreateMul(x, y, "xy.mul")
	case token.Div:
		result = b.Builder.CreateSDiv(x, y, "xy.div")
	case token.Mod:
		result = b.Builder.CreateSRem(x, y, "xy.mod")
	}

	for _, block := range program.Blocks {
		b.emitBlock(block)
	}

	b.Builder.CreateRet(result)

	if ok := llvm.VerifyModule(b.Mod, llvm.ReturnStatusAction); ok != nil {
		fmt.Println(ok.Error())
	}
	b.Mod.Dump()

	engine, err := llvm.NewExecutionEngine(b.Mod)
	if err != nil {
		fmt.Println(err.Error())
	}

	funcResult := engine.RunFunction(b.Main, []llvm.GenericValue{})
	fmt.Printf("%d\n", funcResult.Int(false))
}

func (b *builder) emitBlock(block *ir.BasicBlock) {
	// lb := b.Ctx.AddBasicBlock(b.Main, block.Name()) // how to create unattached basic block?
	idents := make(map[*ir.Val]llvm.Value)
	if block.Stack.Access > 0 {
		b.callStackCheck(block.Stack.Access)
	}
	stackLen := b.Builder.CreateLoad(b.StackLen, "stack_len_"+block.Name())
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
	// return lb
}

func (b *builder) callStackCheck(access int) {
	// accessConst := llvm.ConstInt(llvm.Int64Type(), uint64(access), false)
	// b.StackLen
}
