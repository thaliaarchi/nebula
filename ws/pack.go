package ws // import "github.com/andrewarchi/nebula/ws"

import "errors"

type packer struct {
	text   []byte
	bits   []byte
	curr   byte
	offset uint
	bit    uint
}

// ErrUnmatchedBit is returned from Unpack when a 1 without a matching
// bit is the last bit.
var ErrUnmatchedBit = errors.New("unpack: unmatched 1 bit")

// Pack bit packs a Whitespace source. If the final instruction does
// not end in LF, the end instruction is appended.
func Pack(src []byte) []byte {
	p := packer{src, nil, 0, 0, 7}
	for {
		c, eof := p.readByte()
		if eof {
			// Append end instruction to allow zero trimming.
			if len(src) == 0 || src[len(src)-1] != lf {
				p.writeBit(1)
				p.writeBit(1)
				p.writeBit(1)
				p.writeBit(1)
				p.writeBit(1)
				p.writeBit(1)
			}
			if p.bit != 7 {
				p.bits = append(p.bits, p.curr)
			}
			return p.bits
		}
		switch c {
		case space:
			p.writeBit(0)
		case tab:
			p.writeBit(1)
			p.writeBit(0)
		case lf:
			p.writeBit(1)
			p.writeBit(1)
		}
	}
}

// Unpack expands a bit packed source. The final instruction must
// end in LF.
func Unpack(bits []byte) ([]byte, error) {
	p := packer{nil, bits, 0, 0, 7}
	for {
		b, eof := p.readBit()
		if eof {
			return p.text, nil
		}
		if !b {
			p.writeByte(space)
		} else {
			b, eof = p.readBit()
			if eof {
				return nil, ErrUnmatchedBit
			}
			if b {
				p.writeByte(lf)
			} else {
				p.writeByte(tab)
			}
		}
	}
}

func (p *packer) readByte() (byte, bool) {
	if p.offset >= uint(len(p.text)) {
		return 0, true
	}
	c := p.text[p.offset]
	p.offset++
	return c, false
}

func (p *packer) writeByte(c byte) {
	p.text = append(p.text, c)
}

func (p *packer) readBit() (bool, bool) {
	if p.offset >= uint(len(p.bits)) {
		return false, true
	}
	c := p.bits[p.offset]
	// Ignore trailing zeros on last byte as all
	// control flow instructions end with LF (11).
	if p.offset+1 == uint(len(p.bits)) && c<<(7-p.bit) == 0 {
		return false, true
	}
	b := c&(1<<p.bit) != 0
	if p.bit == 0 {
		p.bit = 7
		p.offset++
	} else {
		p.bit--
	}
	return b, false
}

func (p *packer) writeBit(b byte) {
	p.curr |= b << p.bit
	if p.bit == 0 {
		p.bit = 7
		p.bits = append(p.bits, p.curr)
		p.curr = 0
	} else {
		p.bit--
	}
}
