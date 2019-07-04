package ws

import "testing"

func TestInstrTypeGroups(t *testing.T) {
	tests := []struct {
		IsStack, IsArith, IsHeap, IsFlow, IsIO bool
		Instrs                                 []InstrType
	}{
		{true, false, false, false, false, []InstrType{Push, Dup, Copy, Swap, Drop, Slide}},
		{false, true, false, false, false, []InstrType{Add, Sub, Mul, Div, Mod}},
		{false, false, true, false, false, []InstrType{Store, Retrieve}},
		{false, false, false, true, false, []InstrType{Label, Call, Jmp, Jz, Jn, Ret, End}},
		{false, false, false, false, true, []InstrType{Printc, Printi, Readc, Readi}},
	}

	for _, test := range tests {
		for _, instr := range test.Instrs {
			if instr.IsStack() != test.IsStack {
				t.Errorf("(%s).IsStack() = %t, want %t", &instr, instr.IsStack(), test.IsStack)
			}
			if instr.IsArith() != test.IsArith {
				t.Errorf("(%s).IsArith() = %t, want %t", &instr, instr.IsArith(), test.IsArith)
			}
			if instr.IsHeap() != test.IsHeap {
				t.Errorf("(%s).IsHeap() = %t, want %t", &instr, instr.IsHeap(), test.IsHeap)
			}
			if instr.IsFlow() != test.IsFlow {
				t.Errorf("(%s).IsFlow() = %t, want %t", &instr, instr.IsFlow(), test.IsFlow)
			}
			if instr.IsIO() != test.IsIO {
				t.Errorf("(%s).IsIO() = %t, want %t", &instr, instr.IsIO(), test.IsIO)
			}
		}
	}
}
