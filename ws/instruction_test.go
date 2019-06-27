package ws

import "testing"

func TestInstrTypeGroups(t *testing.T) {
	tests := []struct {
		Group  InstrType
		Instrs []InstrType
	}{
		{StackInstr, []InstrType{Push, Dup, Copy, Swap, Drop, Slide}},
		{ArithInstr, []InstrType{Add, Sub, Mul, Div, Mod}},
		{HeapInstr, []InstrType{Store, Retrieve}},
		{FlowInstr, []InstrType{Label, Call, Jmp, Jz, Jn, Ret, End}},
		{IOInstr, []InstrType{Printc, Printi, Readc, Readi}},
	}
	groups := []InstrType{StackInstr, ArithInstr, HeapInstr, FlowInstr, IOInstr}

	for _, test := range tests {
		for _, instr := range test.Instrs {
			if instr&test.Group == 0 {
				t.Errorf("Instruction %s is not in group %s", &instr, &test.Group)
			}
			for _, group := range groups {
				if group != test.Group && instr&group != 0 {
					t.Errorf("Instruction %s is in incorrect group %s", &instr, &group)
				}
			}
		}
	}
}
