package optimize

import (
	"go/token"
	"math/big"
	"reflect"
	"testing"

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

	tokens := []*ws.Token{
		{Type: ws.Push, Arg: big.NewInt(1), Pos: 1, End: 1},     // 1
		{Type: ws.Push, Arg: big.NewInt(3), Pos: 2, End: 2},     // 2
		{Type: ws.Push, Arg: big.NewInt(10), Pos: 3, End: 3},    // 3
		{Type: ws.Push, Arg: big.NewInt(2), Pos: 4, End: 4},     // 4
		{Type: ws.Mul, Pos: 5, End: 5},                          // 5
		{Type: ws.Add, Pos: 6, End: 6},                          // 6
		{Type: ws.Swap, Pos: 7, End: 7},                         // 7
		{Type: ws.Push, Arg: big.NewInt('C'), Pos: 8, End: 8},   // 8
		{Type: ws.Dup, Pos: 9, End: 9},                          // 9
		{Type: ws.Copy, Arg: big.NewInt(2), Pos: 10, End: 10},   // 10
		{Type: ws.Sub, Pos: 11, End: 11},                        // 11
		{Type: ws.Push, Arg: big.NewInt(-32), Pos: 12, End: 12}, // 12
		{Type: ws.Push, Arg: big.NewInt('a'), Pos: 13, End: 13}, // 13
		{Type: ws.Add, Pos: 14, End: 14},                        // 14
		{Type: ws.Printc, Pos: 15, End: 15},                     // 15
		{Type: ws.Printc, Pos: 16, End: 16},                     // 16
		{Type: ws.Printc, Pos: 17, End: 17},                     // 17
		{Type: ws.Printi, Pos: 18, End: 18},                     // 18
		{Type: ws.Printi, Pos: 19, End: 19},                     // 19
	}
	file := token.NewFileSet().AddFile("test", -1, 0)
	p := &ws.Program{File: file, Tokens: tokens}

	var (
		push1     = ir.NewIntConst(big.NewInt(1), 1)
		push3     = ir.NewIntConst(big.NewInt(3), 2)
		push10    = ir.NewIntConst(big.NewInt(10), 3)
		push2     = ir.NewIntConst(big.NewInt(2), 4)
		mul       = ir.NewBinaryExpr(ir.Mul, push10, push2, 5)
		add1      = ir.NewBinaryExpr(ir.Add, push3, mul, 6)
		pushC     = ir.NewIntConst(big.NewInt('C'), 8)
		sub       = ir.NewBinaryExpr(ir.Sub, pushC, push1, 11)
		pushn32   = ir.NewIntConst(big.NewInt(-32), 12)
		pusha     = ir.NewIntConst(big.NewInt('a'), 13)
		add2      = ir.NewBinaryExpr(ir.Add, pushn32, pusha, 14)
		printAdd2 = ir.NewPrintStmt(ir.PrintByte, add2, 15)
		flushAdd2 = ir.NewFlushStmt(15)
		printSub  = ir.NewPrintStmt(ir.PrintByte, sub, 16)
		flushSub  = ir.NewFlushStmt(16)
		printC    = ir.NewPrintStmt(ir.PrintByte, pushC, 17)
		flushC    = ir.NewFlushStmt(17)
		print1    = ir.NewPrintStmt(ir.PrintInt, push1, 18)
		flush1    = ir.NewFlushStmt(18)
		printAdd1 = ir.NewPrintStmt(ir.PrintInt, add1, 19)
		flushAdd1 = ir.NewFlushStmt(19)
	)

	var stack ir.Stack
	stack.Push(push1)   // 0
	stack.Push(push3)   // 1
	stack.Push(push10)  // 2
	stack.Push(push2)   // 3
	stack.Pop(4)        // 4
	stack.Pop(4)        // 4
	stack.Push(mul)     // 4
	stack.Pop(5)        // 5
	stack.Pop(5)        // 5
	stack.Push(add1)    // 5
	stack.Swap(6)       // 6
	stack.Push(pushC)   // 7
	stack.Dup(8)        // 8
	stack.Copy(2, 9)    // 9
	stack.Pop(10)       // 10
	stack.Pop(10)       // 10
	stack.Push(sub)     // 10
	stack.Push(pushn32) // 11
	stack.Push(pusha)   // 12
	stack.Pop(13)       // 13
	stack.Pop(13)       // 13
	stack.Push(add2)    // 13
	stack.Pop(14)       // 14
	stack.Pop(15)       // 15
	stack.Pop(16)       // 16
	stack.Pop(17)       // 17
	stack.Pop(18)       // 18

	if stack.Len() != 0 || stack.Pops() != 0 || stack.Accesses() != 0 {
		t.Errorf("stack should be empty and not underflow, got %v", stack)
	}

	blockStart := &ir.BasicBlock{
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
		NextBlockID: 1,
		File:        file,
	}

	program, err := p.LowerIR()
	if err != nil {
		t.Errorf("unexpected parse error: %v", err)
	}
	if !reflect.DeepEqual(program, programStart) {
		t.Errorf("SSA conversion not equal\ngot:\n%v\nwant:\n%v", program, programStart)
	}

	var (
		fold20 = ir.NewIntConst(big.NewInt(20), 5)
		fold23 = ir.NewIntConst(big.NewInt(23), 6)
		foldB  = ir.NewIntConst(big.NewInt('B'), 11)
		foldA  = ir.NewIntConst(big.NewInt('A'), 14)
	)

	mul.ReplaceUsesWith(fold20)
	mul.ClearOperands()
	add1.ReplaceUsesWith(fold23)
	add1.ClearOperands()
	sub.ReplaceUsesWith(foldB)
	sub.ClearOperands()
	add2.ReplaceUsesWith(foldA)
	add2.ClearOperands()

	blockConst := &ir.BasicBlock{
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
		NextBlockID: 1,
		File:        file,
	}

	FoldConstArith(program)
	if !reflect.DeepEqual(program, programConst) {
		t.Errorf("constant arithmetic folding not equal\ngot:\n%v\nwant:\n%v", program, programConst)
	}
}
