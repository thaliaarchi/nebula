package analysis // import "github.com/andrewarchi/nebula/analysis"

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/andrewarchi/nebula/bigint"
	"github.com/andrewarchi/nebula/ir"
	"github.com/andrewarchi/nebula/ws"
)

func TestJoinSimpleEntries(t *testing.T) {
	tokens := []ws.Token{
		{Type: ws.Push, Arg: big.NewInt(1)},  // 0
		{Type: ws.Add},                       // 1
		{Type: ws.Mul},                       // 2
		{Type: ws.Label, Arg: big.NewInt(1)}, // 3
		{Type: ws.Copy, Arg: big.NewInt(5)},  // 4
		{Type: ws.Mod},                       // 5
		{Type: ws.Slide, Arg: big.NewInt(2)}, // 6
	}

	p := &ws.Program{Name: "test", Tokens: tokens, LabelNames: nil}
	program, err := p.ConvertSSA()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	v1 := ir.Val(&ir.ConstVal{Val: big.NewInt(1)})
	s0 := ir.Val(&ir.StackVal{Val: 0})
	s1 := ir.Val(&ir.StackVal{Val: 1})
	s2 := ir.Val(&ir.StackVal{Val: 2})
	sn1 := ir.Val(&ir.StackVal{Val: -1})
	sn2 := ir.Val(&ir.StackVal{Val: -2})
	sn7 := ir.Val(&ir.StackVal{Val: -7})

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
			&ir.ArithExpr{Op: ir.Add, Assign: &s0, LHS: &sn1, RHS: &v1},
			&ir.ArithExpr{Op: ir.Mul, Assign: &s1, LHS: &sn2, RHS: &s0},
			&ir.ArithExpr{Op: ir.Mod, Assign: &s2, LHS: &s1, RHS: &sn7},
		},
		Terminator: &ir.ExitStmt{},
		Entries:    []*ir.BasicBlock{nil},
		Callers:    []*ir.BasicBlock{nil},
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
