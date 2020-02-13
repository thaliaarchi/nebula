package ws // import "github.com/andrewarchi/nebula/ws"

import "testing"

func TestUnpack(t *testing.T) {
	src := []byte{0b00101001, 0b01101111, 0b10111001, 0b11111000} // padded with three 1s
	text, err := Unpack(src)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{space, space, tab, tab, space, tab, lf,
		space, lf, lf, tab, lf, tab, space, lf, lf, lf}
	if string(text) != string(want) {
		t.Errorf("got %q, want %q", text, want)
	}
}
