package ast

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/andrewarchi/wspace/token"
)

func TestMergeSimpleCalls(t *testing.T) {
	tokens := []token.Token{
		{Type: token.Label, Arg: big.NewInt(0)}, // 0
		{Type: token.Push, Arg: big.NewInt(1)},  // 1
		{Type: token.Add},                       // 2
		{Type: token.Mul},                       // 3
		{Type: token.Label, Arg: big.NewInt(1)}, // 4
		{Type: token.Copy, Arg: big.NewInt(5)},  // 5
		{Type: token.Mod},                       // 6
		{Type: token.Slide, Arg: big.NewInt(2)}, // 7
	}

	ast, err := Parse(tokens)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var stack Stack
	s1 := stack.PushConst(big.NewInt(1)) // 1
	stack.Pop()                          // 2
	stack.Pop()                          // 2
	s2 := stack.Push()                   // 2
	stack.Pop()                          // 3
	stack.Pop()                          // 3
	s3 := stack.Push()                   // 3
	stack.Copy(5)                        // 5
	stack.Pop()                          // 6
	stack.Pop()                          // 6
	s5 := stack.Push()                   // 6
	stack.Slide(2)                       // 7
	n1 := Val(&StackVal{-1})
	n2 := Val(&StackVal{-2})
	n7 := Val(&StackVal{-7})

	astMerged := &AST{Blocks: []*BasicBlock{{
		Labels: []*big.Int{big.NewInt(0)},
		Nodes: []Node{
			&AssignStmt{Assign: s2, Expr: &ArithExpr{Op: token.Add, LHS: &n1, RHS: s1}},
			&AssignStmt{Assign: s3, Expr: &ArithExpr{Op: token.Mul, LHS: &n2, RHS: s2}},
			&AssignStmt{Assign: s5, Expr: &ArithExpr{Op: token.Mod, LHS: s3, RHS: &n7}},
		},
		Exit:  &EndStmt{},
		Stack: stack,
	}}}

	ast.MergeSimpleCalls()
	if !reflect.DeepEqual(ast, astMerged) {
		t.Errorf("merge not equal\ngot:\n%v\nwant:\n%v", ast, astMerged)
	}
}
