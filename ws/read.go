package ws // import "github.com/andrewarchi/nebula/ws"

import (
	"bufio"
	"io"

	"github.com/icza/bitio"
)

type SpaceToken uint8

const (
	EOF SpaceToken = iota
	Space
	Tab
	LF
)

func (tok SpaceToken) String() string {
	switch tok {
	case EOF:
		return "EOF"
	case Space:
		return "Space"
	case Tab:
		return "Tab"
	case LF:
		return "LF"
	}
	return "illegal"
}

type SpaceReader interface {
	Next() (SpaceToken, error)
	Pos() Pos
}

type TextReader struct {
	br       io.ByteReader
	filename string
	line     int
	col      int
}

func NewTextReader(r io.Reader, filename string) *TextReader {
	var br io.ByteReader
	br, ok := r.(io.ByteReader)
	if !ok {
		br = bufio.NewReader(r)
	}
	return &TextReader{br, filename, 1, 1}
}

func (l *TextReader) Next() (SpaceToken, error) {
	for {
		b, err := l.br.ReadByte()
		if err == io.EOF {
			return EOF, nil
		}
		if err != nil {
			return EOF, err
		}
		l.col++
		switch b {
		case ' ':
			return Space, nil
		case '\t':
			return Tab, nil
		case '\n':
			l.line++
			l.col = 1
			return LF, nil
		}
	}
}

func (l *TextReader) Pos() Pos {
	return Pos{l.filename, l.line, l.col}
}

type BitReader struct {
	br       bitio.Reader
	filename string
	pos      int
}

func NewBitReader(r io.Reader, filename string) *BitReader {
	var br bitio.Reader
	br, ok := r.(bitio.Reader)
	if !ok {
		br = bitio.NewReader(r)
	}
	return &BitReader{br, filename, 0}
}

func (l *BitReader) Next() (SpaceToken, error) {
	b, err := l.br.ReadBool()
	if err == io.EOF {
		return EOF, nil
	}
	if err != nil {
		return EOF, err
	}
	l.pos++
	if !b {
		return Space, nil
	}
	b, err = l.br.ReadBool()
	if err == io.EOF {
		return EOF, nil
	}
	if err != nil {
		return EOF, err
	}
	l.pos++
	if b {
		return LF, nil
	}
	return Tab, nil
}

func (l *BitReader) Pos() Pos {
	return Pos{l.filename, (l.pos / 8) + 1, (l.pos % 8) + 1}
}
