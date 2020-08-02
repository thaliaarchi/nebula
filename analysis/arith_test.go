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
	// push 1    ; 1
	// push 3    ; 2
	// push 10   ; 3
	// push 2    ; 4
	// mul       ; 5
	// add       ; 6
	// swap      ; 7
	// push 'C'  ; 8
	// dup       ; 9
	// copy 2    ; 10
	// sub       ; 11
	// push -32  ; 12
	// push 'a'  ; 13
	// add       ; 14
	// printc    ; 15
	// printc    ; 16
	// printc    ; 17
	// printi    ; 18
	// printi    ; 19

	tokens := []ws.Token{
		{Type: ws.Push, Arg: big.NewInt(1), Start: 1, End: 1},     // 1
		{Type: ws.Push, Arg: big.NewInt(3), Start: 2, End: 2},     // 2
		{Type: ws.Push, Arg: big.NewInt(10), Start: 3, End: 3},    // 3
		{Type: ws.Push, Arg: big.NewInt(2), Start: 4, End: 4},     // 4
		{Type: ws.Mul, Start: 5, End: 5},                          // 5
		{Type: ws.Add, Start: 6, End: 6},                          // 6
		{Type: ws.Swap, Start: 7, End: 7},                         // 7
		{Type: ws.Push, Arg: big.NewInt('C'), Start: 8, End: 8},   // 8
		{Type: ws.Dup, Start: 9, End: 9},                          // 9
		{Type: ws.Copy, Arg: big.NewInt(2), Start: 10, End: 10},   // 10
		{Type: ws.Sub, Start: 11, End: 11},                        // 11
		{Type: ws.Push, Arg: big.NewInt(-32), Start: 12, End: 12}, // 12
		{Type: ws.Push, Arg: big.NewInt('a'), Start: 13, End: 13}, // 13
		{Type: ws.Add, Start: 14, End: 14},                        // 14
		{Type: ws.Printc, Start: 15, End: 15},                     // 15
		{Type: ws.Printc, Start: 16, End: 16},                     // 16
		{Type: ws.Printc, Start: 17, End: 17},                     // 17
		{Type: ws.Printi, Start: 18, End: 18},                     // 18
		{Type: ws.Printi, Start: 19, End: 19},                     // 19
	}
	file := token.NewFileSet().AddFile("test", -1, 0)
	p := &ws.Program{File: file, Tokens: tokens, LabelNames: nil}

	var (
		big1   = big.NewInt(1)
		big3   = big.NewInt(3)
		big10  = big.NewInt(10)
		big2   = big.NewInt(2)
		bigC   = big.NewInt('C')
		bign32 = big.NewInt(-32)
		biga   = big.NewInt('a')

		push1     = ir.NewIntConst(big1, 1)
		push3     = ir.NewIntConst(big3, 2)
		push10    = ir.NewIntConst(big10, 3)
		push2     = ir.NewIntConst(big2, 4)
		mul       = ir.NewBinaryExpr(ir.Mul, push10, push2, 5)
		add1      = ir.NewBinaryExpr(ir.Add, push3, mul, 6)
		pushC     = ir.NewIntConst(bigC, 8)
		sub       = ir.NewBinaryExpr(ir.Sub, pushC, push1, 11)
		pushn32   = ir.NewIntConst(bign32, 12)
		pusha     = ir.NewIntConst(biga, 13)
		add2      = ir.NewBinaryExpr(ir.Add, pushn32, pusha, 14)
		printAdd2 = ir.NewPrintStmt(ir.Printc, add2, 15)
		flushAdd2 = ir.NewFlushStmt(15)
		printSub  = ir.NewPrintStmt(ir.Printc, sub, 16)
		flushSub  = ir.NewFlushStmt(16)
		printC    = ir.NewPrintStmt(ir.Printc, pushC, 17)
		flushC    = ir.NewFlushStmt(17)
		print1    = ir.NewPrintStmt(ir.Printi, push1, 18)
		flush1    = ir.NewFlushStmt(18)
		printAdd1 = ir.NewPrintStmt(ir.Printi, add1, 19)
		flushAdd1 = ir.NewFlushStmt(19)
	)

	constVals := bigint.NewMap()
	constVals.Put(big1, push1)
	constVals.Put(big3, push3)
	constVals.Put(big10, push10)
	constVals.Put(big2, push2)
	constVals.Put(bigC, pushC)
	constVals.Put(bign32, pushn32)
	constVals.Put(biga, pusha)

	var stack ir.Stack
	stack.Push(push1)   // 0
	stack.Push(push3)   // 1
	stack.Push(push10)  // 2
	stack.Push(push2)   // 3
	stack.Pop()         // 4
	stack.Pop()         // 4
	stack.Push(mul)     // 4
	stack.Pop()         // 5
	stack.Pop()         // 5
	stack.Push(add1)    // 5
	stack.Swap()        // 6
	stack.Push(pushC)   // 7
	stack.Dup()         // 8
	stack.Copy(2)       // 9
	stack.Pop()         // 10
	stack.Pop()         // 10
	stack.Push(sub)     // 10
	stack.Push(pushn32) // 11
	stack.Push(pusha)   // 12
	stack.Pop()         // 13
	stack.Pop()         // 13
	stack.Push(add2)    // 13
	stack.Pop()         // 14
	stack.Pop()         // 15
	stack.Pop()         // 16
	stack.Pop()         // 17
	stack.Pop()         // 18

	if len(stack.Vals) != 0 || stack.Pops != 0 || stack.Access != 0 {
		t.Errorf("stack should be empty and not underflow, got %v", stack)
	}

	blockStart := &ir.BasicBlock{
		Stack: stack,
		Nodes: []ir.Inst{
			mul,
			add1,
			sub,
			add2,
			printAdd2,
			flushAdd2,
			printSub,
			flushSub,
			printC,
			flushC,
			print1,
			flush1,
			printAdd1,
			flushAdd1,
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
		File:        file,
	}
	// stack.LoadHandler = blockStart.AppendNode

	program, err := p.ConvertSSA()
	if err != nil {
		t.Errorf("unexpected parse error: %v", err)
	}
	loadHandler := program.Entry.Stack.LoadHandler
	program.Entry.Stack.LoadHandler = nil // for equality

	if !reflect.DeepEqual(program, programStart) {
		t.Errorf("SSA conversion not equal\ngot:\n%v\nwant:\n%v", program, programStart)
	}

	var (
		big20 = big.NewInt(20)
		big23 = big.NewInt(23)
		bigA  = big.NewInt('A')
		bigB  = big.NewInt('B')

		fold20 = ir.NewIntConst(big20, 5)
		fold23 = ir.NewIntConst(big23, 6)
		foldB  = ir.NewIntConst(bigB, 11)
		foldA  = ir.NewIntConst(bigA, 14)
	)

	constVals.Put(big.NewInt(20), fold20)
	constVals.Put(big.NewInt(23), fold23)
	constVals.Put(big.NewInt('A'), foldA)
	constVals.Put(big.NewInt('B'), foldB)

	ir.ReplaceUses(mul, fold20)
	ir.ClearOperands(mul)
	ir.ReplaceUses(add1, fold23)
	ir.ClearOperands(add1)
	ir.ReplaceUses(sub, foldB)
	ir.ClearOperands(sub)
	ir.ReplaceUses(add2, foldA)
	ir.ClearOperands(add2)

	blockConst := &ir.BasicBlock{
		Stack: stack,
		Nodes: []ir.Inst{
			printAdd2,
			flushAdd2,
			printSub,
			flushSub,
			printC,
			flushC,
			print1,
			flush1,
			printAdd1,
			flushAdd1,
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
		File:        file,
	}
	// stack.LoadHandler = blockConst.AppendNode

	program.Entry.Stack.LoadHandler = loadHandler
	FoldConstArith(program)
	program.Entry.Stack.LoadHandler = nil // for equality

	if !reflect.DeepEqual(program, programConst) {
		t.Errorf("constant arithmetic folding not equal\ngot:\n%v\nwant:\n%v", program, programConst)
	}

	/*var (
		strABC123   = ir.NewStringConst("ABC123", 18)
		printABC123 = ir.NewPrintStmt(ir.Prints, strABC123, 18)
	)

	blockStr := &ir.BasicBlock{
		Nodes: []ir.Inst{
			printABC123,
			flushAdd1,
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
