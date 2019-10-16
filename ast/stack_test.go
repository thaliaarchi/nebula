package ast // import "github.com/andrewarchi/nebula/ast"

import (
	"reflect"
	"testing"
)

var (
	vn4 = makeVal(-4)
	vn3 = makeVal(-3)
	vn2 = makeVal(-2)
	vn1 = makeVal(-1)
	v0  = makeVal(0)
	v1  = makeVal(1)
	v2  = makeVal(2)
	v3  = makeVal(3)
)

type stackValTest struct {
	Stack *Stack
	Want  *Stack
	Val   *Val
}

type stackNTest struct {
	Stack *Stack
	Want  *Stack
	N     int
}

type stackTest struct {
	Stack *Stack
	Want  *Stack
}

func TestPush(t *testing.T) {
	for i, test := range []stackValTest{
		{&Stack{nil, nil, 0, 0}, &Stack{[]*Val{v0}, nil, 0, 0}, v0},
		{&Stack{[]*Val{v0, v1}, []*Val{vn1}, 0, 0}, &Stack{[]*Val{v0, v1, v3}, []*Val{vn1}, 0, 0}, v3},
	} {
		test.Stack.Push(test.Val)
		stackEqual(t, i, test.Stack, test.Want)
	}
}

func TestPop(t *testing.T) {
	for i, test := range []stackValTest{
		{&Stack{nil, nil, 0, 0}, &Stack{nil, []*Val{vn1}, 1, 1}, vn1},
		{&Stack{nil, []*Val{nil, nil, vn3}, 3, 7}, &Stack{nil, []*Val{nil, nil, vn3, vn4}, 4, 7}, vn4},
		{&Stack{[]*Val{v0, v1}, nil, 0, 0}, &Stack{[]*Val{v0}, nil, 0, 0}, v1},
	} {
		valEqual(t, i, test.Stack.Pop(), test.Val)
		stackEqual(t, i, test.Stack, test.Want)
	}
}

func TestPopN(t *testing.T) {
	for i, test := range []stackNTest{
		{&Stack{nil, nil, 0, 0}, &Stack{nil, nil, 1, 1}, 1},
		{&Stack{nil, nil, 3, 7}, &Stack{nil, nil, 5, 7}, 2},
		{&Stack{[]*Val{v0, v1}, nil, 0, 0}, &Stack{[]*Val{v0}, nil, 0, 0}, 1},
		{&Stack{[]*Val{v0, v1}, nil, 0, 0}, &Stack{[]*Val{}, nil, 2, 2}, 4},
		{&Stack{[]*Val{v0, v1}, nil, 0, 0}, &Stack{[]*Val{v0, v1}, nil, 0, 0}, 0},
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
		{&Stack{nil, nil, 0, 0}, &Stack{[]*Val{vn1, vn2}, []*Val{vn1, vn2}, 2, 2}},
		{&Stack{nil, nil, 2, 7}, &Stack{[]*Val{vn3, vn4}, []*Val{nil, nil, vn3, vn4}, 4, 7}},
		{&Stack{[]*Val{v0, v1}, nil, 0, 0}, &Stack{[]*Val{v1, v0}, nil, 0, 0}},
		{&Stack{[]*Val{v2}, nil, 1, 1}, &Stack{[]*Val{v2, vn2}, []*Val{nil, vn2}, 2, 2}},

		{&Stack{[]*Val{vn1, vn2}, []*Val{vn1, vn2}, 2, 2}, &Stack{[]*Val{}, []*Val{vn1, vn2}, 0, 2}},
		{&Stack{[]*Val{vn3, vn4}, []*Val{nil, nil, vn3, vn4}, 4, 7}, &Stack{[]*Val{}, []*Val{nil, nil, vn3, vn4}, 2, 7}},
		{&Stack{[]*Val{v1, v0}, nil, 0, 0}, &Stack{[]*Val{v0, v1}, nil, 0, 0}},
		{&Stack{[]*Val{v2, vn2}, []*Val{nil, vn2}, 2, 2}, &Stack{[]*Val{v2}, []*Val{nil, vn2}, 1, 2}},
	} {
		test.Stack.Swap()
		stackEqual(t, i, test.Stack, test.Want)
	}
}

func TestSimplify(t *testing.T) {
	for i, test := range []stackTest{
		{&Stack{nil, nil, 0, 0}, &Stack{nil, nil, 0, 0}},
		{&Stack{[]*Val{v0, v1}, nil, 0, 0}, &Stack{[]*Val{v0, v1}, nil, 0, 0}},
		{&Stack{[]*Val{vn1, vn2}, []*Val{vn1, vn2}, 1, 2}, &Stack{[]*Val{vn2}, []*Val{vn1, vn2}, 0, 2}},
		{&Stack{[]*Val{vn3, vn2, v0}, []*Val{nil, vn2, vn3}, 3, 3}, &Stack{[]*Val{v0}, []*Val{nil, vn2, vn3}, 1, 3}},
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

func valEqual(t *testing.T, testIndex int, got, want *Val) {
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

func makeVal(n int) *Val {
	val := Val(&StackVal{n})
	return &val
}
