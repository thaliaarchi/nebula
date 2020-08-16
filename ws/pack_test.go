package ws

import "testing"

var tests = []struct{ unpacked, packed []byte }{
	{ // no marker bit, no padding
		[]byte{lf, space, tab, space, lf},
		[]byte{0b11010011},
	},
	{ // no marker bit, padding
		[]byte{space, space, tab, space, tab, tab, lf},
		[]byte{0b00100101, 0b01100000},
	},
	{ // marker bit, no padding
		[]byte{space, space, tab, tab, space, tab, lf, space, lf, lf, tab, lf, tab, space},
		[]byte{0b00101001, 0b01101111, 0b10111001},
	},
	{ // marker bit, padding
		[]byte{space, space, tab, tab, lf, tab, space, space, tab},
		[]byte{0b00101011, 0b10001010},
	},
}

func TestPack(t *testing.T) {
	for i, test := range tests {
		p := Pack(test.unpacked)
		if string(p) != string(test.packed) {
			t.Errorf("test %d: got %b, want %b", i, p, test.packed)
		}
	}
}

func TestUnpack(t *testing.T) {
	for i, test := range tests {
		u := Unpack(test.packed)
		if string(u) != string(test.unpacked) {
			t.Errorf("test %d: got %b, want %b", i, u, test.unpacked)
		}
	}
}
