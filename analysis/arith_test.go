package analysis // import "github.com/andrewarchi/nebula/analysis"

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/andrewarchi/nebula/bigint"
	"github.com/andrewarchi/nebula/ir"
	"github.com/andrewarchi/nebula/ws"
)

func TestTransforms(t *testing.T) {
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

	v1 := ir.Val(&ir.ConstVal{Int: big.NewInt(1)})
	v2 := ir.Val(&ir.ConstVal{Int: big.NewInt(2)})
	v3 := ir.Val(&ir.ConstVal{Int: big.NewInt(3)})
	v10 := ir.Val(&ir.ConstVal{Int: big.NewInt(10)})
	v20 := ir.Val(&ir.ConstVal{Int: big.NewInt(20)})
	v23 := ir.Val(&ir.ConstVal{Int: big.NewInt(23)})
	vn32 := ir.Val(&ir.ConstVal{Int: big.NewInt(-32)})
	vA := ir.Val(&ir.ConstVal{Int: big.NewInt('A')})
	vB := ir.Val(&ir.ConstVal{Int: big.NewInt('B')})
	vC := ir.Val(&ir.ConstVal{Int: big.NewInt('C')})
	va := ir.Val(&ir.ConstVal{Int: big.NewInt('a')})
	vABC123 := ir.Val(&ir.StringVal{Str: "ABC123"})
	s0 := ir.Val(&ir.SSAVal{})
	s1 := ir.Val(&ir.SSAVal{})
	s2 := ir.Val(&ir.SSAVal{})
	s3 := ir.Val(&ir.SSAVal{})

	var stack ir.Stack
	stack.Push(&v1)   // 0
	stack.Push(&v3)   // 1
	stack.Push(&v10)  // 2
	stack.Push(&v2)   // 3
	stack.Pop()       // 4
	stack.Pop()       // 4
	stack.Push(&s0)   // 4
	stack.Pop()       // 5
	stack.Pop()       // 5
	stack.Push(&s1)   // 5
	stack.Swap()      // 6
	stack.Push(&vC)   // 7
	stack.Dup()       // 8
	stack.Copy(2)     // 9
	stack.Pop()       // 10
	stack.Pop()       // 10
	stack.Push(&s2)   // 10
	stack.Push(&vn32) // 11
	stack.Push(&va)   // 12
	stack.Pop()       // 13
	stack.Pop()       // 13
	stack.Push(&s3)   // 13
	stack.Pop()       // 14
	stack.Pop()       // 15
	stack.Pop()       // 16
	stack.Pop()       // 17
	stack.Pop()       // 18

	if len(stack.Vals) != 0 || stack.Pops != 0 || stack.Access != 0 {
		t.Errorf("stack should be empty and not underflow, got %v", stack)
	}

	constVals := bigint.NewMap()
	constVals.Put(big.NewInt(1), &v1)
	constVals.Put(big.NewInt(3), &v3)
	constVals.Put(big.NewInt(10), &v10)
	constVals.Put(big.NewInt(2), &v2)
	constVals.Put(big.NewInt('C'), &vC)
	constVals.Put(big.NewInt(2), &v2)
	constVals.Put(big.NewInt(-32), &vn32)
	constVals.Put(big.NewInt('a'), &va)

	blockStart := &ir.BasicBlock{
		Stack: stack,
		Nodes: []ir.Node{
			&ir.BinaryExpr{Op: ir.Mul, Assign: &s0, LHS: &v10, RHS: &v2},
			&ir.BinaryExpr{Op: ir.Add, Assign: &s1, LHS: &v3, RHS: &s0},
			&ir.BinaryExpr{Op: ir.Sub, Assign: &s2, LHS: &vC, RHS: &v1},
			&ir.BinaryExpr{Op: ir.Add, Assign: &s3, LHS: &vn32, RHS: &va},
			&ir.PrintStmt{Op: ir.Printc, Val: &s3},
			&ir.PrintStmt{Op: ir.Printc, Val: &s2},
			&ir.PrintStmt{Op: ir.Printc, Val: &vC},
			&ir.PrintStmt{Op: ir.Printi, Val: &v1},
			&ir.PrintStmt{Op: ir.Printi, Val: &s1},
		},
		Terminator: &ir.ExitStmt{},
		Entries:    []*ir.BasicBlock{nil},
		Callers:    []*ir.BasicBlock{nil},
	}
	programStart := &ir.Program{
		Name:        "test",
		Blocks:      []*ir.BasicBlock{blockStart},
		Entry:       blockStart,
		ConstVals:   *constVals,
		NextBlockID: 1,
	}

	p := &ws.Program{Name: "test", Tokens: tokens, LabelNames: nil}
	program, err := p.ConvertSSA()
	if err != nil {
		t.Errorf("unexpected parse error: %v", err)
	}
	if !reflect.DeepEqual(program, programStart) {
		t.Errorf("token parse not equal\ngot:\n%v\nwant:\n%v", program, programStart)
	}

	constVals.Put(big.NewInt('A'), &vA)
	constVals.Put(big.NewInt('B'), &vB)
	constVals.Put(big.NewInt(20), &v20)
	constVals.Put(big.NewInt(23), &v23)

	blockConst := &ir.BasicBlock{
		Stack: stack,
		Nodes: []ir.Node{
			&ir.PrintStmt{Op: ir.Printc, Val: &vA},
			&ir.PrintStmt{Op: ir.Printc, Val: &vB},
			&ir.PrintStmt{Op: ir.Printc, Val: &vC},
			&ir.PrintStmt{Op: ir.Printi, Val: &v1},
			&ir.PrintStmt{Op: ir.Printi, Val: &v23},
		},
		Terminator: &ir.ExitStmt{},
		Entries:    []*ir.BasicBlock{nil},
		Callers:    []*ir.BasicBlock{nil},
	}
	programConst := &ir.Program{
		Name:        "test",
		Blocks:      []*ir.BasicBlock{blockConst},
		Entry:       blockConst,
		ConstVals:   *constVals,
		NextBlockID: 1,
	}

	FoldConstArith(program)
	if !reflect.DeepEqual(program, programConst) {
		t.Errorf("constant arithmetic folding not equal\ngot:\n%v\nwant:\n%v", program, programConst)
	}

	blockStr := &ir.BasicBlock{
		Nodes: []ir.Node{
			&ir.PrintStmt{Op: ir.Prints, Val: &vABC123},
		},
		Terminator: &ir.ExitStmt{},
		Stack:      stack,
		Entries:    []*ir.BasicBlock{nil},
		Callers:    []*ir.BasicBlock{nil},
	}
	programStr := &ir.Program{
		Name:        "test",
		Blocks:      []*ir.BasicBlock{blockStr},
		Entry:       blockStr,
		ConstVals:   *constVals,
		NextBlockID: 1,
	}

	ConcatStrings(program)
	if !reflect.DeepEqual(program, programStr) {
		t.Errorf("string concat not equal\ngot:\n%v\nwant:\n%v", program, programStr)
	}
}
