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

/*
func TestPop(t *testing.T) {
	for i, test := range []stackValTest{
		{&Stack{nil, 0, 0, 0}, &Stack{nil, 0, -1, -1}, -1},
		{&Stack{nil, 6, -3, -7}, &Stack{nil, 6, -4, -7}, -4},
		{&Stack{[]int{0, 1}, 3, 0, 0}, &Stack{[]int{0}, 3, 0, 0}, 1},
	} {
		intEqual(t, i, test.Stack.Pop(), test.Val)
		stackEqual(t, i, test.Stack, test.Want)
	}
}

func TestPopN(t *testing.T) {
	for i, test := range []stackValTest{
		{&Stack{nil, 0, 0, 0}, &Stack{nil, 0, -1, -1}, 1},
		{&Stack{nil, 6, -3, -7}, &Stack{nil, 6, -4, -7}, 1},
		{&Stack{[]int{0, 1}, 3, 0, 0}, &Stack{[]int{0}, 3, 0, 0}, 1},
		{&Stack{[]int{0, 1}, 3, 0, 0}, &Stack{[]int{}, 3, -2, -2}, 4},
		{&Stack{[]int{0, 1}, 3, 0, 0}, &Stack{[]int{0, 1}, 3, 0, 0}, 0},
	} {
		test.Stack.PopN(test.Val)
		stackEqual(t, i, test.Stack, test.Want)
	}

	checkPanic(t, -1, "stack: pop count must be positive: -1", func() {
		new(Stack).PopN(-1)
	})
}

func TestSwap(t *testing.T) {
	for i, test := range []stackTest{
		{&Stack{nil, 0, 0, 0}, &Stack{[]int{-1, -2}, 0, -2, -2}},
		{&Stack{nil, 6, -3, -7}, &Stack{[]int{-4, -5}, 6, -5, -7}},
		{&Stack{[]int{0, 1}, 3, 0, 0}, &Stack{[]int{1, 0}, 3, 0, 0}},
		{&Stack{[]int{2}, 3, -1, -1}, &Stack{[]int{2, -2}, 3, -2, -2}},

		{&Stack{[]int{-1, -2}, 0, -2, -2}, &Stack{[]int{}, 0, 0, -2}},
		{&Stack{[]int{-4, -5}, 6, -5, -7}, &Stack{[]int{}, 6, -3, -7}},
		{&Stack{[]int{1, 0}, 3, 0, 0}, &Stack{[]int{0, 1}, 3, 0, 0}},
		{&Stack{[]int{2, -2}, 3, -2, -2}, &Stack{[]int{2}, 3, -1, -2}},
	} {
		test.Stack.Swap()
		stackEqual(t, i, test.Stack, test.Want)
	}
}

func TestSimplify(t *testing.T) {
	for i, test := range []stackTest{
		{&Stack{nil, 0, 0, 0}, &Stack{nil, 0, 0, 0}},
		{&Stack{[]int{0, 1}, 3, 0, 0}, &Stack{[]int{0, 1}, 3, 0, 0}},
		{&Stack{[]int{-1, -2}, 3, -1, -2}, &Stack{[]int{-2}, 3, 0, -2}},
		{&Stack{[]int{-3, -2, 0}, 2, -3, -3}, &Stack{[]int{0}, 2, -1, -3}},
	} {
		test.Stack.simplify()
		stackEqual(t, i, test.Stack, test.Want)
	}
}
*/

func stackEqual(t *testing.T, testIndex int, got, want *Stack) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("test %d: got stack %v, want %v", testIndex, got, want)
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
