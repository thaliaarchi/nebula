package ws // import "github.com/andrewarchi/nebula/ws"

type packer struct {
	text   []byte
	bits   []byte
	curr   byte
	offset uint
	bit    uint
}

// Pack bit packs a Whitespace source.
func Pack(src []byte) []byte {
	p := packer{src, nil, 0, 0, 7}
	appendOne := false
	for {
		c, eof := p.readByte()
		if eof {
			if appendOne { // marker bit follows trailing zeros
				p.writeBit(1)
			}
			p.flushBits()
			return p.bits
		}
		switch c {
		case space:
			p.writeBit(0)
			appendOne = true
		case tab:
			p.writeBit(1)
			p.writeBit(0)
			appendOne = true
		case lf:
			p.writeBit(1)
			p.writeBit(1)
			appendOne = false
		}
	}
}

// Unpack expands a bit packed source.
func Unpack(bits []byte) []byte {
	p := packer{nil, bits, 0, 0, 7}
	for {
		b, eof := p.readBit()
		if eof {
			return p.text
		}
		if !b {
			p.writeByte(space)
		} else {
			b, eof = p.readBit()
			if eof { // marker bit
				return p.text
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
	// Ignore trailing zeros on last byte
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

func (p *packer) flushBits() {
	if p.bit != 7 && p.curr != 0 {
		p.bits = append(p.bits, p.curr)
	}
}
