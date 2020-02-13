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

type stateFn func(*lexer) (stateFn, error)

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
	var err error
	for state := lexInst; state != nil; {
		state, err = state(l)
		if err != nil {
			return nil, err
		}
	}
	return l.tokens, nil
}

func (l *lexer) appendToken(typ Type, arg *big.Int) {
	l.tokens = append(l.tokens, Token{typ, arg, l.pos, Pos{"", l.line, l.col}})
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

func transition(s states) stateFn {
	return func(l *lexer) (stateFn, error) {
		l.pos = Pos{"", l.line, l.col}
		c, eof := l.next()
		if eof {
			if s.Root {
				return nil, nil
			}
			return nil, io.ErrUnexpectedEOF
		}
		switch c {
		case space:
			return s.Space, nil
		case tab:
			return s.Tab, nil
		case lf:
			return s.LF, nil
		default:
			panic("unreachable")
		}
	}
}

func emitInst(typ Type) stateFn {
	return func(l *lexer) (stateFn, error) {
		l.appendToken(typ, nil)
		return lexInst, nil
	}
}

func lexNumber(typ Type) stateFn {
	return func(l *lexer) (stateFn, error) {
		arg, err := l.lexSigned()
		if err != nil {
			return nil, err
		}
		l.appendToken(typ, arg)
		return lexInst, nil
	}
}

func lexLabel(typ Type) stateFn {
	return func(l *lexer) (stateFn, error) {
		arg, err := l.lexUnsigned()
		if err != nil {
			return nil, err
		}
		l.appendToken(typ, arg)
		return lexInst, nil
	}
}

func (l *lexer) lexSigned() (*big.Int, error) {
	tok, eof := l.next()
	if eof {
		return nil, errors.New("unterminated number")
	}
	switch tok {
	case space:
		return l.lexUnsigned()
	case tab:
		num, err := l.lexUnsigned()
		if err != nil {
			return nil, err
		}
		num.Neg(num)
		return num, nil
	case lf:
		return bigZero, nil
	default:
		panic("unreachable")
	}
}

func (l *lexer) lexUnsigned() (*big.Int, error) {
	num := new(big.Int)
	for {
		tok, eof := l.next()
		if eof {
			return nil, fmt.Errorf("unterminated number: %d", num)
		}
		switch tok {
		case space:
			num.Lsh(num, 1)
		case tab:
			num.Lsh(num, 1).Or(num, bigOne)
		case lf:
			return num, nil
		default:
			panic("unreachable")
		}
	}
}

func init() {
	lexInst = transition(states{
		Space: lexStack,
		Tab: transition(states{
			Space: lexArith,
			Tab:   lexHeap,
			LF:    lexIO,
		}),
		LF:   lexFlow,
		Root: true,
	})
}

var lexInst stateFn

var lexStack = transition(states{
	Space: lexNumber(Push),
	Tab: transition(states{
		Space: lexNumber(Copy),
		LF:    lexNumber(Slide),
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
		Space: lexLabel(Label),
		Tab:   lexLabel(Call),
		LF:    lexLabel(Jmp),
	}),
	Tab: transition(states{
		Space: lexLabel(Jz),
		Tab:   lexLabel(Jn),
		LF:    emitInst(Ret),
	}),
	LF: transition(states{
		LF: emitInst(End),
	}),
})
