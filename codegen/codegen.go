package codegen

import (
	"fmt"

	"github.com/andrewarchi/nebula/ir"
	"github.com/andrewarchi/nebula/token"
	"llvm.org/llvm/bindings/go/llvm"
)

const maxStackSize = 1024

func EmitLLVMIR(p *ir.Program) {
	builder := llvm.NewBuilder()
	mod := llvm.NewModule(p.Name)

	main := llvm.FunctionType(llvm.Int64Type(), []llvm.Type{}, false)
	llvm.AddFunction(mod, "main", main)
	block := llvm.AddBasicBlock(mod.NamedFunction("main"), "entry")
	builder.SetInsertPoint(block, block.FirstInstruction())

	stack := builder.CreateAlloca(llvm.ArrayType(llvm.Int64Type(), maxStackSize), "stack") // should be global

	zero := llvm.ConstInt(llvm.Int64Type(), 0, false)
	aIdx := llvm.ConstInt(llvm.Int64Type(), 5, false)
	bIdx := llvm.ConstInt(llvm.Int64Type(), 4, false)
	aGep := builder.CreateInBoundsGEP(stack, []llvm.Value{zero, aIdx}, "a.gep")
	bGep := builder.CreateInBoundsGEP(stack, []llvm.Value{zero, bIdx}, "b.gep")
	builder.CreateStore(llvm.ConstInt(llvm.Int64Type(), 16, false), aGep)
	builder.CreateStore(llvm.ConstInt(llvm.Int64Type(), 42, false), bGep)
	a := builder.CreateLoad(aGep, "a")
	b := builder.CreateLoad(bGep, "b")

	op := token.Add
	var result llvm.Value
	switch op {
	case token.Add:
		result = builder.CreateAdd(a, b, "ab")
	case token.Sub:
		result = builder.CreateSub(a, b, "ab")
	case token.Mul:
		result = builder.CreateMul(a, b, "ab")
	case token.Div:
		result = builder.CreateSDiv(a, b, "ab")
	case token.Mod:
		result = builder.CreateSRem(a, b, "ab")
	}

	builder.CreateRet(result)

	if ok := llvm.VerifyModule(mod, llvm.ReturnStatusAction); ok != nil {
		fmt.Println(ok.Error())
	}
	mod.Dump()

	engine, err := llvm.NewExecutionEngine(mod)
	if err != nil {
		fmt.Println(err.Error())
	}

	funcResult := engine.RunFunction(mod.NamedFunction("main"), []llvm.GenericValue{})
	fmt.Printf("%d\n", funcResult.Int(false))
}
