package smt // import "github.com/andrewarchi/nebula/smt"

import (
	"testing"

	"github.com/mitchellh/go-z3"
)

func TestSMTSolve(t *testing.T) {
	config := z3.NewConfig()
	defer config.Close()

	ctx := z3.NewContext(config)
	defer ctx.Close()

	x := ctx.Const(ctx.Symbol("x"), ctx.IntSort())
	v10 := ctx.Int(10, ctx.IntSort())
	v7 := ctx.Int(7, ctx.IntSort())
	v0 := ctx.Int(0, ctx.IntSort())

	eq := x.Mul(v10).Sub(v7).Eq(v0)

	s := ctx.NewSolver()
	defer s.Close()
	s.Assert(eq)

	result := s.Check()
	if result != z3.True {
		t.Fatalf("bad: %s", result)
	}

	m := s.Model()
	defer m.Close()
	t.Logf("\nModel:\n%s", m.String())
}
