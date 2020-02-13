package ws // import "github.com/andrewarchi/nebula/ws"

import "errors"

type unpacker struct {
	src    []byte
	text   []byte
	offset uint
	bit    uint
}

// Unpack expands a bit packed source. The final instruction must be a
// control flow instruction.
func Unpack(src []byte) ([]byte, error) {
	u := unpacker{src, nil, 0, 7}
	for {
		b, eof := u.readBit()
		if eof {
			return u.text, nil
		}
		if !b {
			u.write(space)
		} else {
			b, eof = u.readBit()
			if eof {
				return nil, errors.New("unpack: unmatched 1")
			}
			if b {
				u.write(lf)
			} else {
				u.write(tab)
			}
		}
	}
}

func (u *unpacker) readBit() (bool, bool) {
	if u.offset >= uint(len(u.src)) {
		return false, true
	}
	c := u.src[u.offset]
	// Ignore trailing zeros on last byte as all
	// control flow instructions end with LF (11).
	if u.offset+1 == uint(len(u.src)) && c<<(7-u.bit) == 0 {
		return false, true
	}
	b := c&(1<<u.bit) != 0
	if u.bit == 0 {
		u.bit = 7
		u.offset++
	} else {
		u.bit--
	}
	return b, false
}

func (u *unpacker) write(c byte) {
	u.text = append(u.text, c)
}
