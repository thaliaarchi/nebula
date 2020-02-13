package ws // import "github.com/andrewarchi/nebula/ws"

import (
	"errors"
	"fmt"
	"io"
	"math/big"
)

type lexer struct {
	src    []byte
	tokens []Token
	line   int
	col    int
	offset int
	pos    Pos
}

type stateFn func(*lexer) error

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

var (
	bigZero = big.NewInt(0)
	bigOne  = big.NewInt(1)
)

// Lex lexically analyzes a Whitespace source to produce tokens.
func Lex(src []byte) ([]Token, error) {
	l := &lexer{
		src:    src,
		line:   1,
		col:    1,
		offset: 0,
		pos:    Pos{"", 1, 1},
	}
	for {
		err := lexInst(l)
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			return l.tokens, nil
		}
	}
}

func (l *lexer) next() (byte, bool) {
	for l.offset < len(l.src) {
		c := l.src[l.offset]
		l.offset++
		switch c {
		case space, tab:
			l.col++
			return c, false
		case lf:
			l.line++
			l.col = 1
			return c, false
		default:
			l.col++
		}
	}
	return 0, true
}

func (l *lexer) appendToken(typ Type, arg *big.Int) error {
	l.tokens = append(l.tokens, Token{typ, arg, l.pos, Pos{"", l.line, l.col}})
	return nil
}

func transition(s states) stateFn {
	return func(l *lexer) error {
		l.pos = Pos{"", l.line, l.col}
		c, eof := l.next()
		if eof {
			if s.Root {
				return io.EOF
			}
			return io.ErrUnexpectedEOF
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
			return errors.New("invalid instruction") // TODO report instruction string
		}
		return state(l)
	}
}

func lexNumber(typ Type, signed bool) stateFn {
	return func(l *lexer) error {
		var negative bool
		if signed {
			tok, eof := l.next()
			if eof {
				return errors.New("unterminated number")
			}
			switch tok {
			case space:
			case tab:
				negative = true
			case lf:
				return l.appendToken(typ, bigZero)
			default:
				panic("unreachable")
			}
		}
		num := new(big.Int)
		for {
			tok, eof := l.next()
			if eof {
				return fmt.Errorf("unterminated number: %d", num)
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
				return l.appendToken(typ, num)
			default:
				panic("unreachable")
			}
		}
	}
}

func emitInst(typ Type) stateFn {
	return func(l *lexer) error {
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
