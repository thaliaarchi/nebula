package ast

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/andrewarchi/wspace/token"
)

func TestTransforms(t *testing.T) {
	var stack Stack
	vA, sA := &ConstVal{new(big.Int).SetInt64('A')}, &StackVal{stack.Push()}
	vB, sB := &ConstVal{new(big.Int).SetInt64('B')}, &StackVal{stack.Push()}
	vC, sC := &ConstVal{new(big.Int).SetInt64('C')}, &StackVal{stack.Push()}
	v1, s1 := &ConstVal{new(big.Int).SetInt64(1)}, &StackVal{stack.Push()}
	v23, s23 := &ConstVal{new(big.Int).SetInt64(23)}, &StackVal{stack.Push()}

	ast := AST{&BasicBlock{
		Nodes: []Node{
			&UnaryExpr{Op: token.Push, Assign: s23, Val: v23},
			&UnaryExpr{Op: token.Push, Assign: s1, Val: v1},
			&UnaryExpr{Op: token.Push, Assign: sC, Val: vC},
			&UnaryExpr{Op: token.Push, Assign: sB, Val: vB},
			&UnaryExpr{Op: token.Push, Assign: sA, Val: vA},
			&PrintStmt{Op: token.Printc, Val: sA},
			&PrintStmt{Op: token.Printc, Val: sB},
			&PrintStmt{Op: token.Printc, Val: sC},
			&PrintStmt{Op: token.Printi, Val: s1},
			&PrintStmt{Op: token.Printi, Val: s23},
		},
		Edge:  &EndStmt{},
		Stack: stack,
	}}

	astConst := AST{&BasicBlock{
		Nodes: []Node{
			&PrintStmt{Op: token.Printc, Val: vA},
			&PrintStmt{Op: token.Printc, Val: vB},
			&PrintStmt{Op: token.Printc, Val: vC},
			&PrintStmt{Op: token.Printi, Val: v1},
			&PrintStmt{Op: token.Printi, Val: v23},
		},
		Edge:  &EndStmt{},
		Stack: stack,
	}}

	astStr := AST{&BasicBlock{
		Nodes: []Node{
			&PrintStmt{Op: token.Prints, Val: &StringVal{"ABC123"}},
		},
		Edge:  &EndStmt{},
		Stack: stack,
	}}

	ast.InlineStackConstants()
	if !reflect.DeepEqual(ast, astConst) {
		t.Errorf("constant inlining not equal\ngot:\n%v\nwant:\n%v", ast, astConst)
	}
	ast.ConcatStrings()
	if !reflect.DeepEqual(ast, astStr) {
		t.Errorf("string concat not equal\ngot:\n%v\nwant:\n%v", ast, astStr)
	}
}
