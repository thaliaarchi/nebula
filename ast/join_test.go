package ast // import "github.com/andrewarchi/nebula/ast"

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/andrewarchi/nebula/token"
)

func TestJoinSimpleCalls(t *testing.T) {
	tokens := []token.Token{
		{Type: token.Push, Arg: big.NewInt(1)},  // 0
		{Type: token.Add},                       // 1
		{Type: token.Mul},                       // 2
		{Type: token.Label, Arg: big.NewInt(1)}, // 3
		{Type: token.Copy, Arg: big.NewInt(5)},  // 4
		{Type: token.Mod},                       // 5
		{Type: token.Slide, Arg: big.NewInt(2)}, // 6
	}

	ast, err := Parse(tokens, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var stack Stack
	s1 := stack.PushConst(big.NewInt(1)) // 0
	stack.Pop()                          // 1
	stack.Pop()                          // 1
	s2 := stack.Push()                   // 1
	stack.Pop()                          // 2
	stack.Pop()                          // 2
	s3 := stack.Push()                   // 2
	stack.Copy(5)                        // 4
	stack.Pop()                          // 5
	stack.Pop()                          // 5
	s5 := stack.Push()                   // 5
	stack.Slide(2)                       // 6
	n1 := Val(&StackVal{-1})
	n2 := Val(&StackVal{-2})
	n7 := Val(&StackVal{-7})

	blockJoined := &BasicBlock{
		Stack: stack,
		Nodes: []Node{
			&AssignStmt{Assign: s2, Expr: &ArithExpr{Op: token.Add, LHS: &n1, RHS: s1}},
			&AssignStmt{Assign: s3, Expr: &ArithExpr{Op: token.Mul, LHS: &n2, RHS: s2}},
			&AssignStmt{Assign: s5, Expr: &ArithExpr{Op: token.Mod, LHS: s3, RHS: &n7}},
		},
		Terminator: &EndStmt{},
		Entries:    []*BasicBlock{entryBlock},
		Callers:    []*BasicBlock{entryBlock},
	}
	astJoined := &AST{
		Blocks: []*BasicBlock{blockJoined},
		Entry:  blockJoined,
		NextID: 2,
	}

	ast.JoinSimpleCalls()
	if !reflect.DeepEqual(ast, astJoined) {
		t.Errorf("join not equal\ngot:\n%v\nwant:\n%v", ast, astJoined)
	}
}
