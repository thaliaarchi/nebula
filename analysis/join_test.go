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
	file := token.NewFileSet().AddFile("test", -1, 0)
	p := &ws.Program{File: file, Tokens: tokens, LabelNames: nil}

	program, err := p.ConvertSSA()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	c1 := &ir.ConstVal{Int: big.NewInt(1)}
	constVals := bigint.NewMap()
	constVals.Put(big.NewInt(1), &c1)

	load1 := &ir.LoadStackExpr{Def: &ir.ValueDef{}, Pos: 1}
	add := &ir.BinaryExpr{Def: &ir.ValueDef{}, Op: ir.Add}
	ir.AddUse(load1, add, 0)
	ir.AddUse(c1, add, 1)
	load2 := &ir.LoadStackExpr{Def: &ir.ValueDef{}, Pos: 2}
	mul := &ir.BinaryExpr{Def: &ir.ValueDef{}, Op: ir.Mul}
	ir.AddUse(load2, mul, 0)
	ir.AddUse(add, mul, 1)
	load7 := &ir.LoadStackExpr{Def: &ir.ValueDef{}, Pos: 7}
	mod := &ir.BinaryExpr{Def: &ir.ValueDef{}, Op: ir.Mod}
	ir.AddUse(mul, mod, 0)
	ir.AddUse(load7, mod, 1)

	var stack ir.Stack
	stack.Push(c1)  // 0
	stack.Pop()     // 1
	stack.Pop()     // 1
	stack.Push(add) // 1
	stack.Pop()     // 2
	stack.Pop()     // 2
	stack.Push(mul) // 2
	stack.Copy(5)   // 4
	stack.Pop()     // 5
	stack.Pop()     // 5
	stack.Push(mod) // 5
	stack.Slide(2)  // 6

	blockJoined := &ir.BasicBlock{
		Stack: stack,
		Nodes: []ir.Node{
			&ir.CheckStackStmt{Access: 7},
			load1,
			add,
			load2,
			mul,
			load7,
			mod,
		},
		Terminator: &ir.ExitTerm{},
		Entries:    []*ir.BasicBlock{nil},
		Callers:    []*ir.BasicBlock{nil},
	}
	programJoined := &ir.Program{
		Name:        "test",
		Blocks:      []*ir.BasicBlock{blockJoined},
		Entry:       blockJoined,
		ConstVals:   constVals,
		NextBlockID: 1,
	}
	stack.Handler = blockJoined

	JoinSimpleEntries(program)
	if !reflect.DeepEqual(program, programJoined) {
		t.Errorf("join not equal\ngot:\n%v\nwant:\n%v", program, programJoined)
	}
	t.Fatal("JoinSimpleEntries currently broken from restructure")
}
