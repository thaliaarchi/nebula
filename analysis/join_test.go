package analysis // import "github.com/andrewarchi/nebula/analysis"

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/andrewarchi/nebula/bigint"
	"github.com/andrewarchi/nebula/ir"
	"github.com/andrewarchi/nebula/token"
)

func TestJoinSimpleEntries(t *testing.T) {
	tokens := []token.Token{
		{Type: token.Push, Arg: big.NewInt(1)},  // 0
		{Type: token.Add},                       // 1
		{Type: token.Mul},                       // 2
		{Type: token.Label, Arg: big.NewInt(1)}, // 3
		{Type: token.Copy, Arg: big.NewInt(5)},  // 4
		{Type: token.Mod},                       // 5
		{Type: token.Slide, Arg: big.NewInt(2)}, // 6
	}

	program, err := ir.Parse(tokens, nil, "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	v1 := ir.Val(&ir.ConstVal{big.NewInt(1)})
	s0 := ir.Val(&ir.StackVal{0})
	s1 := ir.Val(&ir.StackVal{1})
	s2 := ir.Val(&ir.StackVal{2})
	sn1 := ir.Val(&ir.StackVal{-1})
	sn2 := ir.Val(&ir.StackVal{-2})
	sn7 := ir.Val(&ir.StackVal{-7})

	var stack ir.Stack
	stack.Push(&v1) // 0
	stack.Pop()     // 1
	stack.Pop()     // 1
	stack.Push(&s0) // 1
	stack.Pop()     // 2
	stack.Pop()     // 2
	stack.Push(&s1) // 2
	stack.Copy(5)   // 4
	stack.Pop()     // 5
	stack.Pop()     // 5
	stack.Push(&s2) // 5
	stack.Slide(2)  // 6

	constVals := bigint.NewMap(nil)
	constVals.Put(big.NewInt(1), &v1)

	blockJoined := &ir.BasicBlock{
		Stack: stack,
		Nodes: []ir.Node{
			&ir.AssignStmt{Assign: &s0, Expr: &ir.ArithExpr{Op: token.Add, LHS: &sn1, RHS: &v1}},
			&ir.AssignStmt{Assign: &s1, Expr: &ir.ArithExpr{Op: token.Mul, LHS: &sn2, RHS: &s0}},
			&ir.AssignStmt{Assign: &s2, Expr: &ir.ArithExpr{Op: token.Mod, LHS: &s1, RHS: &sn7}},
		},
		Terminator: &ir.EndStmt{},
		Entries:    []*ir.BasicBlock{ir.EntryBlock},
		Callers:    []*ir.BasicBlock{ir.EntryBlock},
	}
	programJoined := &ir.Program{
		Name:        "test",
		Blocks:      []*ir.BasicBlock{blockJoined},
		Entry:       blockJoined,
		ConstVals:   *constVals,
		NextBlockID: 2,
		NextStackID: 3,
	}

	JoinSimpleEntries(program)
	if !reflect.DeepEqual(program, programJoined) {
		t.Errorf("join not equal\ngot:\n%v\nwant:\n%v", program, programJoined)
	}
}
