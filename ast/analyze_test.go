package ast

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/andrewarchi/wspace/token"
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
		{Type: token.Mul, Arg: nil},              // 4
		{Type: token.Add, Arg: nil},              // 5
		{Type: token.Swap, Arg: nil},             // 6
		{Type: token.Push, Arg: big.NewInt('C')}, // 7
		{Type: token.Dup, Arg: nil},              // 8
		{Type: token.Copy, Arg: big.NewInt(2)},   // 9
		{Type: token.Sub, Arg: nil},              // 10
		{Type: token.Push, Arg: big.NewInt(-32)}, // 11
		{Type: token.Push, Arg: big.NewInt('a')}, // 12
		{Type: token.Add, Arg: nil},              // 13
		{Type: token.Printc, Arg: nil},           // 14
		{Type: token.Printc, Arg: nil},           // 15
		{Type: token.Printc, Arg: nil},           // 16
		{Type: token.Printi, Arg: nil},           // 17
		{Type: token.Printi, Arg: nil},           // 18
	}

	var stack Stack
	s0 := stack.PushConst(big.NewInt(1))    // 0
	s1 := stack.PushConst(big.NewInt(3))    // 1
	s2 := stack.PushConst(big.NewInt(10))   // 2
	s3 := stack.PushConst(big.NewInt(2))    // 3
	stack.Pop()                             // 4
	stack.Pop()                             // 4
	s4 := stack.Push()                      // 4
	stack.Pop()                             // 5
	stack.Pop()                             // 5
	s5 := stack.Push()                      // 5
	stack.Swap()                            // 6
	s7 := stack.PushConst(big.NewInt('C'))  // 7
	stack.Dup()                             // 8
	stack.Copy(2)                           // 9
	stack.Pop()                             // 10
	stack.Pop()                             // 10
	s10 := stack.Push()                     // 10
	s11 := stack.PushConst(big.NewInt(-32)) // 11
	s12 := stack.PushConst(big.NewInt('a')) // 12
	stack.Pop()                             // 13
	stack.Pop()                             // 13
	s13 := stack.Push()                     // 13
	stack.Pop()                             // 14
	stack.Pop()                             // 15
	stack.Pop()                             // 16
	stack.Pop()                             // 17
	stack.Pop()                             // 18

	if len(stack.Vals) != 0 || stack.Low != 0 || stack.Min != 0 {
		t.Errorf("stack should be empty and not underflow, got %v", stack)
	}

	astStart := AST{&BasicBlock{
		Nodes: []Node{
			&AssignStmt{Assign: s4, Expr: &ArithExpr{Op: token.Mul, LHS: s2, RHS: s3}},
			&AssignStmt{Assign: s5, Expr: &ArithExpr{Op: token.Add, LHS: s1, RHS: s4}},
			&AssignStmt{Assign: s10, Expr: &ArithExpr{Op: token.Sub, LHS: s7, RHS: s0}},
			&AssignStmt{Assign: s13, Expr: &ArithExpr{Op: token.Add, LHS: s11, RHS: s12}},
			&PrintStmt{Op: token.Printc, Val: s13},
			&PrintStmt{Op: token.Printc, Val: s10},
			&PrintStmt{Op: token.Printc, Val: s7},
			&PrintStmt{Op: token.Printi, Val: s0},
			&PrintStmt{Op: token.Printi, Val: s5},
		},
		Edge:  &EndStmt{},
		Stack: stack,
	}}

	ast, err := Parse(tokens)
	if err != nil {
		t.Errorf("unexpected parse error: %v", err)
	}
	if !reflect.DeepEqual(ast, astStart) {
		t.Errorf("token parse not equal\ngot:\n%v\nwant:\n%v", ast, astStart)
	}

	vA := &ConstVal{big.NewInt('A')}
	vB := &ConstVal{big.NewInt('B')}
	v23 := &ConstVal{big.NewInt(23)}
	astConst := AST{&BasicBlock{
		Nodes: []Node{
			&PrintStmt{Op: token.Printc, Val: vA},
			&PrintStmt{Op: token.Printc, Val: vB},
			&PrintStmt{Op: token.Printc, Val: s7},
			&PrintStmt{Op: token.Printi, Val: s0},
			&PrintStmt{Op: token.Printi, Val: v23},
		},
		Edge:  &EndStmt{},
		Stack: stack,
	}}

	ast.FoldConstArith()
	if !reflect.DeepEqual(ast, astConst) {
		t.Errorf("constant arithmetic folding not equal\ngot:\n%v\nwant:\n%v", ast, astConst)
	}

	astStr := AST{&BasicBlock{
		Nodes: []Node{
			&PrintStmt{Op: token.Prints, Val: &StringVal{"ABC123"}},
		},
		Edge:  &EndStmt{},
		Stack: stack,
	}}

	ast.ConcatStrings()
	if !reflect.DeepEqual(ast, astStr) {
		t.Errorf("string concat not equal\ngot:\n%v\nwant:\n%v", ast, astStr)
	}
}
