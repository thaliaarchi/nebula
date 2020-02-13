package ws // import "github.com/andrewarchi/nebula/ws"

import (
	"errors"
	"fmt"
	"go/token"
	"io"
	"math/big"
)

// Lexer is a lexical analyzer for Whitespace source.
type Lexer struct {
	file        *token.File
	src         []byte
	offset      int
	startOffset int
	endOffset   int
}

type stateFn func(*Lexer) (*Token, error)

type states struct {
	Space stateFn
	Tab   stateFn
	LF    stateFn
	Root  bool
}

const (
	space = ' '
	tab   = '\t'
	lf    = '\n'
)

// NewLexer constructs a Lexer scan a Whitespace source
// file into tokens.
func NewLexer(file *token.File, src []byte) *Lexer {
	return &Lexer{
		file:        file,
		src:         src,
		offset:      0,
		startOffset: 0,
		endOffset:   0,
	}
}

// Lex scans a single Whitespace token.
func (l *Lexer) Lex() (*Token, error) { // TODO return position of error
	return lexInst(l)
}

// LexProgram scans a Whitespace source file into a Program.
func (l *Lexer) LexProgram() (*Program, error) {
	var tokens []Token
	for {
		tok, err := l.Lex()
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			return &Program{l.file, tokens, nil}, nil
		}
		tokens = append(tokens, *tok)
	}
}

func (l *Lexer) next() (byte, int, bool) {
	for l.offset < len(l.src) {
		offset := l.offset
		l.offset++
		switch c := l.src[offset]; c {
		case space, tab:
			return c, offset, false
		case lf:
			l.file.AddLine(offset)
			return c, offset, false
		}
	}
	return 0, 0, true
}

func (l *Lexer) appendToken(typ Type, arg *big.Int) (*Token, error) {
	return &Token{
		Type:  typ,
		Arg:   arg,
		Start: l.file.Pos(l.startOffset),
		End:   l.file.Pos(l.endOffset),
	}, nil
}

func transition(s states) stateFn {
	return func(l *Lexer) (*Token, error) {
		c, offset, eof := l.next()
		if eof {
			if s.Root {
				return nil, io.EOF
			}
			return nil, io.ErrUnexpectedEOF
		}
		if s.Root {
			l.startOffset = offset
		}
		var state stateFn
		switch c {
		case space:
			state = s.Space
		case tab:
			state = s.Tab
		case lf:
			state = s.LF
		default:
			panic("unreachable")
		}
		if state == nil {
			return nil, errors.New("invalid instruction") // TODO report instruction string
		}
		return state(l)
	}
}

var (
	bigZero = big.NewInt(0)
	bigOne  = big.NewInt(1)
)

func lexNumber(typ Type, signed bool) stateFn {
	return func(l *Lexer) (*Token, error) {
		var negative bool
		if signed {
			tok, offset, eof := l.next()
			if eof {
				return nil, errors.New("unterminated number")
			}
			switch tok {
			case space:
			case tab:
				negative = true
			case lf:
				l.endOffset = offset
				return l.appendToken(typ, bigZero)
			default:
				panic("unreachable")
			}
		}
		num := new(big.Int)
		for {
			tok, offset, eof := l.next()
			if eof {
				return nil, fmt.Errorf("unterminated number: %d", num)
			}
			switch tok {
			case space:
				num.Lsh(num, 1)
			case tab:
				num.Lsh(num, 1).Or(num, bigOne)
			case lf:
				if negative {
					num.Neg(num)
				}
				l.endOffset = offset
				return l.appendToken(typ, num)
			default:
				panic("unreachable")
			}
		}
	}
}

func emitInst(typ Type) stateFn {
	return func(l *Lexer) (*Token, error) {
		return l.appendToken(typ, nil)
	}
}

var lexInst = transition(states{
	Space: lexStack,
	Tab: transition(states{
		Space: lexArith,
		Tab:   lexHeap,
		LF:    lexIO,
	}),
	LF:   lexFlow,
	Root: true,
})

var lexStack = transition(states{
	Space: lexNumber(Push, true),
	Tab: transition(states{
		Space: lexNumber(Copy, true),
		LF:    lexNumber(Slide, true),
	}),
	LF: transition(states{
		Space: emitInst(Dup),
		Tab:   emitInst(Swap),
		LF:    emitInst(Drop),
	}),
})

var lexArith = transition(states{
	Space: transition(states{
		Space: emitInst(Add),
		Tab:   emitInst(Sub),
		LF:    emitInst(Mul),
	}),
	Tab: transition(states{
		Space: emitInst(Div),
		Tab:   emitInst(Mod),
	}),
})

var lexHeap = transition(states{
	Space: emitInst(Store),
	Tab:   emitInst(Retrieve),
})

var lexIO = transition(states{
	Space: transition(states{
		Space: emitInst(Printc),
		Tab:   emitInst(Printi),
	}),
	Tab: transition(states{
		Space: emitInst(Readc),
		Tab:   emitInst(Readi),
	}),
})

var lexFlow = transition(states{
	Space: transition(states{
		Space: lexNumber(Label, false),
		Tab:   lexNumber(Call, false),
		LF:    lexNumber(Jmp, false),
	}),
	Tab: transition(states{
		Space: lexNumber(Jz, false),
		Tab:   lexNumber(Jn, false),
		LF:    emitInst(Ret),
	}),
	LF: transition(states{
		LF: emitInst(End),
	}),
})
