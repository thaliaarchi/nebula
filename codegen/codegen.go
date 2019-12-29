package codegen

import (
	"fmt"

	"llvm.org/llvm/bindings/go/llvm"
)

func EmitLLVMIR() {
	builder := llvm.NewBuilder()
	mod := llvm.NewModule("my_module")

	main := llvm.FunctionType(llvm.Int32Type(), []llvm.Type{}, false)
	llvm.AddFunction(mod, "main", main)
	block := llvm.AddBasicBlock(mod.NamedFunction("main"), "entry")
	builder.SetInsertPoint(block, block.FirstInstruction())

	a := builder.CreateAlloca(llvm.Int32Type(), "a")
	builder.CreateStore(llvm.ConstInt(llvm.Int32Type(), 32, false), a)

	b := builder.CreateAlloca(llvm.Int32Type(), "b")
	builder.CreateStore(llvm.ConstInt(llvm.Int32Type(), 16, false), b)

	bVal := builder.CreateLoad(b, "b_val")
	aVal := builder.CreateLoad(a, "a_val")
	result := builder.CreateAdd(aVal, bVal, "ab_val")
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
