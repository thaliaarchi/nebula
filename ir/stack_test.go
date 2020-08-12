package ir // import "github.com/andrewarchi/nebula/ir"

import (
	"go/token"
	"math/big"
	"reflect"
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
)

type stackTest struct {
	Stack *Stack
	Want  *Stack
	Value Value
	N     int
}

func TestPush(t *testing.T) {
	for i, test := range []stackTest{
		{
			Stack: &Stack{nil, nil, 0, 0, nil},
			Want:  &Stack{[]Value{v0}, nil, 0, 0, nil},
			Value: v0,
		},
		{
			Stack: &Stack{[]Value{v0, v1}, []Value{load1}, 0, 0, nil},
			Want:  &Stack{[]Value{v0, v1, v3}, []Value{load1}, 0, 0, nil},
			Value: v3,
		},
	} {
		test.Stack.Push(test.Value)
		stackEqual(t, i, test.Stack, test.Want)
	}
}

func TestPop(t *testing.T) {
	for i, test := range []stackTest{
		{
			Stack: &Stack{nil, nil, 0, 0, nil},
			Want:  &Stack{nil, []Value{load1}, 1, 1, nil},
			Value: load1,
		},
		{
			Stack: &Stack{nil, []Value{nil, nil, load3}, 3, 7, nil},
			Want:  &Stack{nil, []Value{nil, nil, load3, load4}, 4, 7, nil},
			Value: load4,
		},
		{
			Stack: &Stack{[]Value{v0, v1}, nil, 0, 0, nil},
			Want:  &Stack{[]Value{v0}, nil, 0, 0, nil},
			Value: v1,
		},
	} {
		valEqual(t, i, test.Stack.Pop(), test.Value)
		stackEqual(t, i, test.Stack, test.Want)
	}
}

func TestPopN(t *testing.T) {
	for i, test := range []stackTest{
		{
			Stack: &Stack{nil, nil, 0, 0, nil},
			Want:  &Stack{nil, nil, 1, 1, nil},
			N:     1,
		},
		{
			Stack: &Stack{nil, nil, 3, 7, nil},
			Want:  &Stack{nil, nil, 5, 7, nil},
			N:     2,
		},
		{
			Stack: &Stack{[]Value{v0, v1}, nil, 0, 0, nil},
			Want:  &Stack{[]Value{v0}, nil, 0, 0, nil},
			N:     1,
		},
		{
			Stack: &Stack{[]Value{v0, v1}, nil, 0, 0, nil},
			Want:  &Stack{[]Value{}, nil, 2, 2, nil},
			N:     4,
		},
		{
			Stack: &Stack{[]Value{v0, v1}, nil, 0, 0, nil},
			Want:  &Stack{[]Value{v0, v1}, nil, 0, 0, nil},
			N:     0,
		},
	} {
		test.Stack.PopN(test.N)
		stackEqual(t, i, test.Stack, test.Want)
	}

	checkPanic(t, -1, "stack: pop count must be positive: -1", func() {
		new(Stack).PopN(-1)
	})
}

func TestSwap(t *testing.T) {
	for i, test := range []stackTest{
		{
			Stack: &Stack{nil, nil, 0, 0, nil},
			Want:  &Stack{[]Value{load1, load2}, []Value{load1, load2}, 2, 2, nil},
		},
		{
			Stack: &Stack{nil, nil, 2, 7, nil},
			Want:  &Stack{[]Value{load3, load4}, []Value{nil, nil, load3, load4}, 4, 7, nil},
		},
		{
			Stack: &Stack{[]Value{v0, v1}, nil, 0, 0, nil},
			Want:  &Stack{[]Value{v1, v0}, nil, 0, 0, nil},
		},
		{
			Stack: &Stack{[]Value{v2}, nil, 1, 1, nil},
			Want:  &Stack{[]Value{v2, load2}, []Value{nil, load2}, 2, 2, nil},
		},

		{
			Stack: &Stack{[]Value{load1, load2}, []Value{load1, load2}, 2, 2, nil},
			Want:  &Stack{[]Value{}, []Value{load1, load2}, 0, 2, nil},
		},
		{
			Stack: &Stack{[]Value{load3, load4}, []Value{nil, nil, load3, load4}, 4, 7, nil},
			Want:  &Stack{[]Value{}, []Value{nil, nil, load3, load4}, 2, 7, nil},
		},
		{
			Stack: &Stack{[]Value{v1, v0}, nil, 0, 0, nil},
			Want:  &Stack{[]Value{v0, v1}, nil, 0, 0, nil},
		},
		{
			Stack: &Stack{[]Value{v2, load2}, []Value{nil, load2}, 2, 2, nil},
			Want:  &Stack{[]Value{v2}, []Value{nil, load2}, 1, 2, nil},
		},
	} {
		test.Stack.Swap()
		stackEqual(t, i, test.Stack, test.Want)
	}
}

func TestSimplify(t *testing.T) {
	for i, test := range []stackTest{
		{
			Stack: &Stack{nil, nil, 0, 0, nil},
			Want:  &Stack{nil, nil, 0, 0, nil},
		},
		{
			Stack: &Stack{[]Value{v0, v1}, nil, 0, 0, nil},
			Want:  &Stack{[]Value{v0, v1}, nil, 0, 0, nil},
		},
		{
			Stack: &Stack{[]Value{load1, load2}, []Value{load1, load2}, 1, 2, nil},
			Want:  &Stack{[]Value{load2}, []Value{load1, load2}, 0, 2, nil},
		},
		{
			Stack: &Stack{[]Value{load3, load2, v0}, []Value{nil, load2, load3}, 3, 3, nil},
			Want:  &Stack{[]Value{v0}, []Value{nil, load2, load3}, 1, 3, nil},
		},
	} {
		test.Stack.simplify()
		stackEqual(t, i, test.Stack, test.Want)
	}
}

func stackEqual(t *testing.T, testIndex int, got, want *Stack) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("test %d: got stack %v, want %v", testIndex, got, want)
	}
}

func valEqual(t *testing.T, testIndex int, got, want Value) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("test %d: got val %v, want %v", testIndex, got, want)
	}
}

func checkPanic(t *testing.T, testIndex int, want interface{}, mightPanic func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if r := recover(); r != want {
			t.Errorf("test %d: got panic %v, want panic %v", testIndex, r, want)
		}
	}()
	mightPanic()
}
