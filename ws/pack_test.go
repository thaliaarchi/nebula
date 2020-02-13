package ws // import "github.com/andrewarchi/nebula/ws"

import "testing"

var (
	unpacked = []byte{space, space, tab, tab, space, tab,
		lf, space, lf, lf, tab, lf, tab, space, lf, lf, lf}
	unpackedNoEnd = []byte{space, space, tab, tab, space, tab,
		lf, space, lf, lf, tab, lf, tab, space}
	packed = []byte{0b00101001, 0b01101111, 0b10111001, 0b11111000} // padded with three 0s

	unpackedShort = []byte{lf, space, tab, space, lf}
	packedShort   = []byte{0b11010011} // no padding

	packedUnmatched = []byte{0b11010101} // unmatched 1
)

func TestPack(t *testing.T) {
	for i, test := range []struct{ unpacked, packed []byte }{
		{unpacked, packed},
		{unpackedNoEnd, packed},
		{unpackedShort, packedShort},
	} {
		p := Pack(test.unpacked)
		if string(p) != string(test.packed) {
			t.Errorf("test %d: got %q, want %q", i, p, test.packed)
		}
	}
}

func TestUnpack(t *testing.T) {
	for i, test := range []struct{ packed, unpacked []byte }{
		{packed, unpacked},
		{packedShort, unpackedShort},
	} {
		u, err := Unpack(test.packed)
		if err != nil {
			t.Fatal(err)
		}
		if string(u) != string(test.unpacked) {
			t.Errorf("test %d: got %q, want %q", i, u, test.unpacked)
		}
	}
	_, err := Unpack(packedUnmatched)
	if err != ErrUnmatchedBit {
		t.Errorf("got %v, want %v", err, ErrUnmatchedBit)
	}
}
