package analysis // import "github.com/andrewarchi/nebula/analysis"

import (
	"go/token"
	"math/big"
	"reflect"
	"testing"

	"github.com/andrewarchi/nebula/bigint"
	"github.com/andrewarchi/nebula/ir"
	"github.com/andrewarchi/nebula/ws"
)

func TestFoldConstArith(t *testing.T) {
	// push 1    ; 0
	// push 3    ; 1
	// push 10   ; 2
	// push 2    ; 3
	// mul       ; 4
	// add       ; 5
	// swap      ; 6
	// push 'C'  ; 7
	// dup       ; 8
	// copy 2    ; 9
	// sub       ; 10
	// push -32  ; 11
	// push 'a'  ; 12
	// add       ; 13
	// printc    ; 14
	// printc    ; 15
	// printc    ; 16
	// printi    ; 17
	// printi    ; 18

	tokens := []ws.Token{
		{Type: ws.Push, Arg: big.NewInt(1)},   // 0
		{Type: ws.Push, Arg: big.NewInt(3)},   // 1
		{Type: ws.Push, Arg: big.NewInt(10)},  // 2
		{Type: ws.Push, Arg: big.NewInt(2)},   // 3
		{Type: ws.Mul},                        // 4
		{Type: ws.Add},                        // 5
		{Type: ws.Swap},                       // 6
		{Type: ws.Push, Arg: big.NewInt('C')}, // 7
		{Type: ws.Dup},                        // 8
		{Type: ws.Copy, Arg: big.NewInt(2)},   // 9
		{Type: ws.Sub},                        // 10
		{Type: ws.Push, Arg: big.NewInt(-32)}, // 11
		{Type: ws.Push, Arg: big.NewInt('a')}, // 12
		{Type: ws.Add},                        // 13
		{Type: ws.Printc},                     // 14
		{Type: ws.Printc},                     // 15
		{Type: ws.Printc},                     // 16
		{Type: ws.Printi},                     // 17
		{Type: ws.Printi},                     // 18
	}
	file := token.NewFileSet().AddFile("test", -1, 0)
	p := &ws.Program{File: file, Tokens: tokens, LabelNames: nil}

	var (
		c1   = &ir.ConstVal{Int: big.NewInt(1)}
		c2   = &ir.ConstVal{Int: big.NewInt(2)}
		c3   = &ir.ConstVal{Int: big.NewInt(3)}
		cn32 = &ir.ConstVal{Int: big.NewInt(-32)}
		c10  = &ir.ConstVal{Int: big.NewInt(10)}
		cC   = &ir.ConstVal{Int: big.NewInt('C')}
		ca   = &ir.ConstVal{Int: big.NewInt('a')}
	)

	constVals := bigint.NewMap()
	constVals.Put(big.NewInt(1), c1)
	constVals.Put(big.NewInt(3), c3)
	constVals.Put(big.NewInt(10), c10)
	constVals.Put(big.NewInt(2), c2)
	constVals.Put(big.NewInt('C'), cC)
	constVals.Put(big.NewInt(2), c2)
	constVals.Put(big.NewInt(-32), cn32)
	constVals.Put(big.NewInt('a'), ca)

	mul := &ir.BinaryExpr{Def: &ir.ValueDef{}, Op: ir.Mul}
	ir.AddUse(c10, mul, 0)
	ir.AddUse(c2, mul, 1)
	add1 := &ir.BinaryExpr{Def: &ir.ValueDef{}, Op: ir.Add}
	ir.AddUse(c3, add1, 0)
	ir.AddUse(mul, add1, 1)
	sub := &ir.BinaryExpr{Def: &ir.ValueDef{}, Op: ir.Sub}
	ir.AddUse(cC, sub, 0)
	ir.AddUse(c1, sub, 1)
	add2 := &ir.BinaryExpr{Def: &ir.ValueDef{}, Op: ir.Add}
	ir.AddUse(cn32, add2, 0)
	ir.AddUse(ca, add2, 1)
	printAdd2 := &ir.PrintStmt{Op: ir.Printc}
	ir.AddUse(add2, printAdd2, 0)
	printSub := &ir.PrintStmt{Op: ir.Printc}
	ir.AddUse(sub, printSub, 0)
	printC := &ir.PrintStmt{Op: ir.Printc}
	ir.AddUse(cC, printC, 0)
	print1 := &ir.PrintStmt{Op: ir.Printi}
	ir.AddUse(c1, print1, 0)
	printAdd1 := &ir.PrintStmt{Op: ir.Printi}
	ir.AddUse(add1, printAdd1, 0)

	var stack ir.Stack
	stack.Push(c1)   // 0
	stack.Push(c3)   // 1
	stack.Push(c10)  // 2
	stack.Push(c2)   // 3
	stack.Pop()      // 4
	stack.Pop()      // 4
	stack.Push(mul)  // 4
	stack.Pop()      // 5
	stack.Pop()      // 5
	stack.Push(add1) // 5
	stack.Swap()     // 6
	stack.Push(cC)   // 7
	stack.Dup()      // 8
	stack.Copy(2)    // 9
	stack.Pop()      // 10
	stack.Pop()      // 10
	stack.Push(sub)  // 10
	stack.Push(cn32) // 11
	stack.Push(ca)   // 12
	stack.Pop()      // 13
	stack.Pop()      // 13
	stack.Push(add2) // 13
	stack.Pop()      // 14
	stack.Pop()      // 15
	stack.Pop()      // 16
	stack.Pop()      // 17
	stack.Pop()      // 18

	if len(stack.Vals) != 0 || stack.Pops != 0 || stack.Access != 0 {
		t.Errorf("stack should be empty and not underflow, got %v", stack)
	}

	blockStart := &ir.BasicBlock{
		Stack: stack,
		Nodes: []ir.Node{
			mul,
			add1,
			sub,
			add2,
			printAdd2,
			printSub,
			printC,
			print1,
			printAdd1,
		},
		Terminator: &ir.ExitTerm{},
		Entries:    []*ir.BasicBlock{nil},
		Callers:    []*ir.BasicBlock{nil},
	}
	programStart := &ir.Program{
		Name:        "test",
		Blocks:      []*ir.BasicBlock{blockStart},
		Entry:       blockStart,
		ConstVals:   constVals,
		NextBlockID: 1,
	}
	stack.Handler = blockStart

	program, err := p.ConvertSSA()
	if err != nil {
		t.Errorf("unexpected parse error: %v", err)
	}
	if !reflect.DeepEqual(program, programStart) {
		t.Errorf("SSA conversion not equal\ngot:\n%v\nwant:\n%v", program, programStart)
	}

	var (
		c20 = &ir.ConstVal{Int: big.NewInt(20)}
		c23 = &ir.ConstVal{Int: big.NewInt(23)}
		cA  = &ir.ConstVal{Int: big.NewInt('A')}
		cB  = &ir.ConstVal{Int: big.NewInt('B')}
	)

	constVals.Put(big.NewInt('A'), cA)
	constVals.Put(big.NewInt('B'), cB)
	constVals.Put(big.NewInt(20), c20)
	constVals.Put(big.NewInt(23), c23)

	printA := &ir.PrintStmt{Op: ir.Printc}
	ir.AddUse(cA, printA, 0)
	printB := &ir.PrintStmt{Op: ir.Printc}
	ir.AddUse(cB, printB, 0)
	print23 := &ir.PrintStmt{Op: ir.Printi}
	ir.AddUse(c23, print23, 0)

	blockConst := &ir.BasicBlock{
		Stack: stack,
		Nodes: []ir.Node{
			printA,
			printB,
			printC,
			print1,
			print23,
		},
		Terminator: &ir.ExitTerm{},
		Entries:    []*ir.BasicBlock{nil},
		Callers:    []*ir.BasicBlock{nil},
	}
	programConst := &ir.Program{
		Name:        "test",
		Blocks:      []*ir.BasicBlock{blockConst},
		Entry:       blockConst,
		ConstVals:   constVals,
		NextBlockID: 1,
	}
	stack.Handler = blockConst

	FoldConstArith(program)
	if !reflect.DeepEqual(program, programConst) {
		t.Errorf("constant arithmetic folding not equal\ngot:\n%v\nwant:\n%v", program, programConst)
	}

	/*cABC123 := &ir.StringVal{Str: "ABC123"}
	printABC123 := &ir.PrintStmt{Op: ir.Prints}
	ir.AddUse(cABC123, printABC123, 0)

	blockStr := &ir.BasicBlock{
		Nodes: []ir.Node{
			printABC123,
		},
		Terminator: &ir.ExitTerm{},
		Stack:      stack,
		Entries:    []*ir.BasicBlock{nil},
		Callers:    []*ir.BasicBlock{nil},
	}
	programStr := &ir.Program{
		Name:        "test",
		Blocks:      []*ir.BasicBlock{blockStr},
		Entry:       blockStr,
		ConstVals:   constVals,
		NextBlockID: 1,
	}

	ConcatStrings(program)
	if !reflect.DeepEqual(program, programStr) {
		t.Errorf("string concat not equal\ngot:\n%v\nwant:\n%v", program, programStr)
	}*/
}
