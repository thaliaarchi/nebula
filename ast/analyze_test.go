package ast // import "github.com/andrewarchi/nebula/ast"

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/andrewarchi/nebula/bigint"
	"github.com/andrewarchi/nebula/token"
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

	tokens := []token.Token{
		{Type: token.Push, Arg: big.NewInt(1)},   // 0
		{Type: token.Push, Arg: big.NewInt(3)},   // 1
		{Type: token.Push, Arg: big.NewInt(10)},  // 2
		{Type: token.Push, Arg: big.NewInt(2)},   // 3
		{Type: token.Mul},                        // 4
		{Type: token.Add},                        // 5
		{Type: token.Swap},                       // 6
		{Type: token.Push, Arg: big.NewInt('C')}, // 7
		{Type: token.Dup},                        // 8
		{Type: token.Copy, Arg: big.NewInt(2)},   // 9
		{Type: token.Sub},                        // 10
		{Type: token.Push, Arg: big.NewInt(-32)}, // 11
		{Type: token.Push, Arg: big.NewInt('a')}, // 12
		{Type: token.Add},                        // 13
		{Type: token.Printc},                     // 14
		{Type: token.Printc},                     // 15
		{Type: token.Printc},                     // 16
		{Type: token.Printi},                     // 17
		{Type: token.Printi},                     // 18
	}

	v1 := Val(&ConstVal{big.NewInt(1)})
	v2 := Val(&ConstVal{big.NewInt(2)})
	v3 := Val(&ConstVal{big.NewInt(3)})
	v10 := Val(&ConstVal{big.NewInt(10)})
	v23 := Val(&ConstVal{big.NewInt(23)})
	vn32 := Val(&ConstVal{big.NewInt(-32)})
	vA := Val(&ConstVal{big.NewInt('A')})
	vB := Val(&ConstVal{big.NewInt('B')})
	vC := Val(&ConstVal{big.NewInt('C')})
	va := Val(&ConstVal{big.NewInt('a')})
	vABC123 := Val(&StringVal{"ABC123"})
	s0 := Val(&StackVal{0})
	s1 := Val(&StackVal{1})
	s2 := Val(&StackVal{2})
	s3 := Val(&StackVal{3})

	var stack Stack
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

	constVals := bigint.NewMap(nil)
	constVals.Put(big.NewInt(1), &v1)
	constVals.Put(big.NewInt(3), &v3)
	constVals.Put(big.NewInt(10), &v10)
	constVals.Put(big.NewInt(2), &v2)
	constVals.Put(big.NewInt('C'), &vC)
	constVals.Put(big.NewInt(2), &v2)
	constVals.Put(big.NewInt(-32), &vn32)
	constVals.Put(big.NewInt('a'), &va)

	blockStart := &BasicBlock{
		Stack: stack,
		Nodes: []Node{
			&AssignStmt{Assign: &s0, Expr: &ArithExpr{Op: token.Mul, LHS: &v10, RHS: &v2}},
			&AssignStmt{Assign: &s1, Expr: &ArithExpr{Op: token.Add, LHS: &v3, RHS: &s0}},
			&AssignStmt{Assign: &s2, Expr: &ArithExpr{Op: token.Sub, LHS: &vC, RHS: &v1}},
			&AssignStmt{Assign: &s3, Expr: &ArithExpr{Op: token.Add, LHS: &vn32, RHS: &va}},
			&PrintStmt{Op: token.Printc, Val: &s3},
			&PrintStmt{Op: token.Printc, Val: &s2},
			&PrintStmt{Op: token.Printc, Val: &vC},
			&PrintStmt{Op: token.Printi, Val: &v1},
			&PrintStmt{Op: token.Printi, Val: &s1},
		},
		Terminator: &EndStmt{},
		Entries:    []*BasicBlock{entryBlock},
		Callers:    []*BasicBlock{entryBlock},
	}
	astStart := &AST{
		Blocks:      []*BasicBlock{blockStart},
		Entry:       blockStart,
		ConstVals:   *constVals,
		NextBlockID: 1,
		NextStackID: 4,
	}

	ast, err := Parse(tokens, nil)
	if err != nil {
		t.Errorf("unexpected parse error: %v", err)
	}
	if !reflect.DeepEqual(ast, astStart) {
		t.Errorf("token parse not equal\ngot:\n%v\nwant:\n%v", ast, astStart)
	}

	blockConst := &BasicBlock{
		Stack: stack,
		Nodes: []Node{
			&PrintStmt{Op: token.Printc, Val: &vA},
			&PrintStmt{Op: token.Printc, Val: &vB},
			&PrintStmt{Op: token.Printc, Val: &vC},
			&PrintStmt{Op: token.Printi, Val: &v1},
			&PrintStmt{Op: token.Printi, Val: &v23},
		},
		Terminator: &EndStmt{},
		Entries:    []*BasicBlock{entryBlock},
		Callers:    []*BasicBlock{entryBlock},
	}
	astConst := &AST{
		Blocks:      []*BasicBlock{blockConst},
		Entry:       blockConst,
		ConstVals:   *constVals,
		NextBlockID: 1,
		NextStackID: 4,
	}

	ast.FoldConstArith()
	if !reflect.DeepEqual(ast, astConst) {
		t.Errorf("constant arithmetic folding not equal\ngot:\n%v\nwant:\n%v", ast, astConst)
	}

	blockStr := &BasicBlock{
		Nodes: []Node{
			&PrintStmt{Op: token.Prints, Val: &vABC123},
		},
		Terminator: &EndStmt{},
		Stack:      stack,
		Entries:    []*BasicBlock{entryBlock},
		Callers:    []*BasicBlock{entryBlock},
	}
	astStr := &AST{
		Blocks:      []*BasicBlock{blockStr},
		Entry:       blockStr,
		ConstVals:   *constVals,
		NextBlockID: 1,
		NextStackID: 4,
	}

	ast.ConcatStrings()
	if !reflect.DeepEqual(ast, astStr) {
		t.Errorf("string concat not equal\ngot:\n%v\nwant:\n%v", ast, astStr)
	}
}
