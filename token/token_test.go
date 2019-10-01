package token

import (
	"math/big"
	"testing"
)

func TestTokenString(t *testing.T) {
	arg := big.NewInt(123)
	tests := []struct {
		Token  *Token
		String string
	}{
		{&Token{Push, arg}, "push 123"},
		{&Token{Push, nil}, "push 0"},
		{&Token{Add, arg}, "add"},
		{&Token{Add, nil}, "add"},
		{&Token{Label, arg}, "label_123:"},
		{&Token{Label, nil}, "label_0:"},
	}

	for i, test := range tests {
		if str := test.Token.String(); str != test.String {
			t.Errorf("test %d: String() = %q, want %q", i+1, str, test.String)
		}
	}
}

func TestTypeGroups(t *testing.T) {
	tests := []struct {
		IsStack, IsArith, IsHeap, IsFlow, IsIO bool
		Types                                  []Type
	}{
		{true, false, false, false, false, []Type{Push, Dup, Copy, Swap, Drop, Slide}},
		{false, true, false, false, false, []Type{Add, Sub, Mul, Div, Mod}},
		{false, false, true, false, false, []Type{Store, Retrieve}},
		{false, false, false, true, false, []Type{Label, Call, Jmp, Jz, Jn, Ret, End}},
		{false, false, false, false, true, []Type{Printc, Printi, Readc, Readi}},
	}

	for _, test := range tests {
		for _, typ := range test.Types {
			if typ.IsStack() != test.IsStack {
				t.Errorf("(%s).IsStack() = %t, want %t", typ, typ.IsStack(), test.IsStack)
			}
			if typ.IsArith() != test.IsArith {
				t.Errorf("(%s).IsArith() = %t, want %t", typ, typ.IsArith(), test.IsArith)
			}
			if typ.IsHeap() != test.IsHeap {
				t.Errorf("(%s).IsHeap() = %t, want %t", typ, typ.IsHeap(), test.IsHeap)
			}
			if typ.IsFlow() != test.IsFlow {
				t.Errorf("(%s).IsFlow() = %t, want %t", typ, typ.IsFlow(), test.IsFlow)
			}
			if typ.IsIO() != test.IsIO {
				t.Errorf("(%s).IsIO() = %t, want %t", typ, typ.IsIO(), test.IsIO)
			}
		}
	}
}

func TestInstrTypeString(t *testing.T) {
	tests := []struct {
		Type   Type
		String string
	}{
		{Push, "push"},
		{Dup, "dup"},
		{Copy, "copy"},
		{Swap, "swap"},
		{Drop, "drop"},
		{Slide, "slide"},
		{Add, "add"},
		{Sub, "sub"},
		{Mul, "mul"},
		{Div, "div"},
		{Mod, "mod"},
		{Store, "store"},
		{Retrieve, "retrieve"},
		{Label, "label"},
		{Call, "call"},
		{Jmp, "jmp"},
		{Jz, "jz"},
		{Jn, "jn"},
		{Ret, "ret"},
		{End, "end"},
		{Fallthrough, "fallthrough"},
		{Printc, "printc"},
		{Printi, "printi"},
		{Readc, "readc"},
		{Readi, "readi"},
		{Illegal, "illegal"},
		{stackBeg, "illegal"},
		{stackEnd, "illegal"},
		{arithBeg, "illegal"},
		{arithEnd, "illegal"},
		{heapBeg, "illegal"},
		{heapEnd, "illegal"},
		{flowBeg, "illegal"},
		{flowEnd, "illegal"},
		{ioBeg, "illegal"},
		{ioEnd, "illegal"},
	}

	for i, test := range tests {
		if str := test.Type.String(); str != test.String {
			t.Errorf("test %d: String() = %q, want %q", i+1, str, test.String)
		}
	}
}
