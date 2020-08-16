package ir

import (
	"fmt"
	"go/token"
	"math/big"
	"testing"
)

var (
	v0    = NewIntConst(big.NewInt(0), 1)
	v1    = NewIntConst(big.NewInt(1), 2)
	v2    = NewIntConst(big.NewInt(2), 3)
	v3    = NewIntConst(big.NewInt(3), 4)
	load1 = NewLoadStackExpr(1, token.NoPos)
	load2 = NewLoadStackExpr(2, token.NoPos)
	load3 = NewLoadStackExpr(3, token.NoPos)
	load4 = NewLoadStackExpr(4, token.NoPos)

	f = NewFormatter()
)

func init() {
	// Register loads with formatter so they are printed consistently
	// across tests.
	f.FormatValue(load1)
	f.FormatValue(load2)
	f.FormatValue(load3)
	f.FormatValue(load4)
}

type stackTest struct {
	Stack *Stack
	Want  *Stack
	Value Value
	N     int
}

func TestPush(t *testing.T) {
	for i, test := range []stackTest{
		{
			Stack: &Stack{nil, nil, 0, 0, handleAccess, handleLoad},
			Want:  &Stack{[]Value{v0}, nil, 0, 0, handleAccess, handleLoad},
			Value: v0,
		},
		{
			Stack: &Stack{[]Value{v0, v1}, []Value{load1}, 0, 0, handleAccess, handleLoad},
			Want:  &Stack{[]Value{v0, v1, v3}, []Value{load1}, 0, 0, handleAccess, handleLoad},
			Value: v3,
		},
	} {
		test.Stack.Push(test.Value)
		checkStack(t, i, test.Stack, test.Want)
	}
}

func TestPop(t *testing.T) {
	for i, test := range []stackTest{
		{
			Stack: &Stack{nil, nil, 0, 0, handleAccess, handleLoad},
			Want:  &Stack{nil, []Value{load1}, 1, 1, handleAccess, handleLoad},
			Value: load1,
		},
		{
			Stack: &Stack{nil, []Value{nil, nil, load3}, 3, 7, handleAccess, handleLoad},
			Want:  &Stack{nil, []Value{nil, nil, load3, load4}, 4, 7, handleAccess, handleLoad},
			Value: load4,
		},
		{
			Stack: &Stack{[]Value{v0, v1}, nil, 0, 0, handleAccess, handleLoad},
			Want:  &Stack{[]Value{v0}, nil, 0, 0, handleAccess, handleLoad},
			Value: v1,
		},
	} {
		checkValue(t, i, test.Stack.Pop(token.NoPos), test.Value)
		checkStack(t, i, test.Stack, test.Want)
	}
}

func TestDropN(t *testing.T) {
	for i, test := range []stackTest{
		{
			Stack: &Stack{nil, nil, 0, 0, handleAccess, handleLoad},
			Want:  &Stack{nil, nil, 1, 1, handleAccess, handleLoad},
			N:     1,
		},
		{
			Stack: &Stack{nil, nil, 3, 7, handleAccess, handleLoad},
			Want:  &Stack{nil, nil, 5, 7, handleAccess, handleLoad},
			N:     2,
		},
		{
			Stack: &Stack{[]Value{v0, v1}, nil, 0, 0, handleAccess, handleLoad},
			Want:  &Stack{[]Value{v0}, nil, 0, 0, handleAccess, handleLoad},
			N:     1,
		},
		{
			Stack: &Stack{[]Value{v0, v1}, nil, 0, 0, handleAccess, handleLoad},
			Want:  &Stack{[]Value{}, nil, 2, 2, handleAccess, handleLoad},
			N:     4,
		},
		{
			Stack: &Stack{[]Value{v0, v1}, nil, 0, 0, handleAccess, handleLoad},
			Want:  &Stack{[]Value{v0, v1}, nil, 0, 0, handleAccess, handleLoad},
			N:     0,
		},
	} {
		test.Stack.DropN(test.N, token.NoPos)
		checkStack(t, i, test.Stack, test.Want)
	}

	checkPanic(t, "stack: drop count must be positive: -1", func() {
		new(Stack).DropN(-1, token.NoPos)
	})
}

func TestSwap(t *testing.T) {
	for i, test := range []stackTest{
		{
			Stack: &Stack{nil, nil, 0, 0, handleAccess, handleLoad},
			Want:  &Stack{[]Value{load1, load2}, []Value{load1, load2}, 2, 2, handleAccess, handleLoad},
		},
		{
			Stack: &Stack{[]Value{load1, load2}, []Value{load1, load2}, 2, 2, handleAccess, handleLoad},
			Want:  &Stack{[]Value{}, []Value{load1, load2}, 0, 2, handleAccess, handleLoad},
		},

		{
			Stack: &Stack{nil, nil, 2, 7, handleAccess, handleLoad},
			Want:  &Stack{[]Value{load3, load4}, []Value{nil, nil, load3, load4}, 4, 7, handleAccess, handleLoad},
		},
		{
			Stack: &Stack{[]Value{load3, load4}, []Value{nil, nil, load3, load4}, 4, 7, handleAccess, handleLoad},
			Want:  &Stack{[]Value{}, []Value{nil, nil, load3, load4}, 2, 7, handleAccess, handleLoad},
		},

		{
			Stack: &Stack{[]Value{v0, v1}, nil, 0, 0, handleAccess, handleLoad},
			Want:  &Stack{[]Value{v1, v0}, nil, 0, 0, handleAccess, handleLoad},
		},
		{
			Stack: &Stack{[]Value{v1, v0}, nil, 0, 0, handleAccess, handleLoad},
			Want:  &Stack{[]Value{v0, v1}, nil, 0, 0, handleAccess, handleLoad},
		},

		{
			Stack: &Stack{[]Value{v2}, nil, 1, 1, handleAccess, handleLoad},
			Want:  &Stack{[]Value{v2, load2}, []Value{nil, load2}, 2, 2, handleAccess, handleLoad},
		},
		{
			Stack: &Stack{[]Value{v2, load2}, []Value{nil, load2}, 2, 2, handleAccess, handleLoad},
			Want:  &Stack{[]Value{v2}, []Value{nil, load2}, 1, 2, handleAccess, handleLoad},
		},
	} {
		test.Stack.Swap(token.NoPos)
		checkStack(t, i, test.Stack, test.Want)
	}
}

func TestSimplify(t *testing.T) {
	for i, test := range []stackTest{
		{
			Stack: &Stack{nil, nil, 0, 0, handleAccess, handleLoad},
			Want:  &Stack{nil, nil, 0, 0, handleAccess, handleLoad},
		},
		{
			Stack: &Stack{[]Value{v0, v1}, nil, 0, 0, handleAccess, handleLoad},
			Want:  &Stack{[]Value{v0, v1}, nil, 0, 0, handleAccess, handleLoad},
		},
		{
			Stack: &Stack{[]Value{load1, load2}, []Value{load1, load2}, 1, 2, handleAccess, handleLoad},
			Want:  &Stack{[]Value{load2}, []Value{load1, load2}, 0, 2, handleAccess, handleLoad},
		},
		{
			Stack: &Stack{[]Value{load3, load2, v0}, []Value{nil, load2, load3}, 3, 3, handleAccess, handleLoad},
			Want:  &Stack{[]Value{v0}, []Value{nil, load2, load3}, 1, 3, handleAccess, handleLoad},
		},
	} {
		test.Stack.simplify()
		checkStack(t, i, test.Stack, test.Want)
	}
}

func handleAccess(n int, pos token.Pos) {
	panic("TODO")
}

func handleLoad(n int, pos token.Pos) Value {
	if n < 1 || n > 4 {
		panic(fmt.Sprintf("handleLoad: n out of range: %d", n))
	}
	return []Value{load1, load2, load3, load4}[n-1]
}

func equals(a, b *Stack) bool {
	if a.pops != b.pops || a.accesses != b.accesses ||
		len(a.values) != len(b.values) || len(a.under) != len(b.under) ||
		(a.values == nil) != (b.values == nil) ||
		(a.under == nil) != (b.under == nil) ||
		(a.HandleLoad == nil) != (b.HandleLoad == nil) {
		return false
	}
	for i := range a.values {
		if a.values[i] != b.values[i] {
			return false
		}
	}
	for i := range a.under {
		if a.under[i] != b.under[i] {
			return false
		}
	}
	return true
}

func checkStack(t *testing.T, testIndex int, got, want *Stack) {
	t.Helper()
	if !equals(got, want) {
		t.Errorf("test %d: got stack %s, want %s", testIndex, f.FormatStack(got), f.FormatStack(want))
	}
}

func checkValue(t *testing.T, testIndex int, got, want Value) {
	t.Helper()
	if got != want {
		t.Errorf("test %d: got value %s, want %s", testIndex, f.FormatValue(got), f.FormatValue(want))
	}
}

func checkPanic(t *testing.T, want interface{}, mightPanic func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if r := recover(); r != want {
			t.Errorf("got panic %v, want panic %v", r, want)
		}
	}()
	mightPanic()
}
