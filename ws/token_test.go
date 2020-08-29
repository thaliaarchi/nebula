package ws

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
		{&Token{Type: Push, Arg: arg}, "push 123"},
		{&Token{Type: Push, Arg: nil}, "push <nil>"},
		{&Token{Type: Add, Arg: arg}, "add"},
		{&Token{Type: Add, Arg: nil}, "add"},
		{&Token{Type: Label, Arg: arg}, "label_123"},
		{&Token{Type: Label, Arg: nil}, "label_<nil>"},
	}

	for i, test := range tests {
		if str := test.Token.String(); str != test.String {
			t.Errorf("test %d: String() = %q, want %q", i+1, str, test.String)
		}
	}
}

func TestTypePredicates(t *testing.T) {
	types := []struct {
		Name      string
		Predicate func(Type) bool
		Types     []Type
	}{
		{"stack", Type.IsStack, []Type{Push, Dup, Copy, Swap, Drop, Slide, Shuffle}},
		{"arith", Type.IsArith, []Type{Add, Sub, Mul, Div, Mod}},
		{"heap", Type.IsHeap, []Type{Store, Retrieve}},
		{"control", Type.IsControl, []Type{Label, Call, Jmp, Jz, Jn, Ret, End}},
		{"io", Type.IsIO, []Type{Printc, Printi, Readc, Readi}},
		{"debug", Type.IsDebug, []Type{Trace, DumpStack, DumpHeap}},
	}

	for i, group := range types {
		for _, typ := range group.Types {
			if !group.Predicate(typ) {
				t.Errorf("%v is not a %s instruction, but should be", typ, group.Name)
			}

			for j, other := range types {
				if i != j && other.Predicate(typ) {
					t.Errorf("%v is a %s instruction, but should not be", typ, other.Name)
				}
			}
		}
	}
}

func TestInstrTypeString(t *testing.T) {
	tests := []struct {
		Type   Type
		String string
	}{
		{Illegal, "token(0)"},
		{Push, "push"},
		{Dup, "dup"},
		{Copy, "copy"},
		{Swap, "swap"},
		{Drop, "drop"},
		{Slide, "slide"},
		{Shuffle, "shuffle"},
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
		{Printc, "printc"},
		{Printi, "printi"},
		{Readc, "readc"},
		{Readi, "readi"},
		{Trace, "trace"},
		{DumpStack, "dumpstack"},
		{DumpHeap, "dumpheap"},
		{100, "token(100)"},
	}

	for i, test := range tests {
		if str := test.Type.String(); str != test.String {
			t.Errorf("test %d: String() = %q, want %q", i+1, str, test.String)
		}
	}
}
