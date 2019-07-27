package ws

import (
	"errors"
	"fmt"
	"io"
	"math/big"
)

type Lexer struct {
	l      SpaceReader
	instrs chan Token
}

func Lex(l SpaceReader) <-chan Token {
	p := &Lexer{
		l:      l,
		instrs: make(chan Token),
	}
	go p.run()
	return p.instrs
}

func (p *Lexer) run() error {
	defer close(p.instrs)
	var err error
	for state := lexInstr; state != nil; {
		state, err = state(p)
		if err != nil {
			return err
		}
	}
	return nil
}

type stateFn func(*Lexer) (stateFn, error)

type states struct {
	Space stateFn
	Tab   stateFn
	LF    stateFn
	Root  bool
}

func transition(s states) stateFn {
	return func(p *Lexer) (stateFn, error) {
		t, err := p.l.Next()
		if err != nil {
			return nil, err
		}
		switch t {
		case Space:
			return s.Space, nil
		case Tab:
			return s.Tab, nil
		case LF:
			return s.LF, nil
		case EOF:
			if !s.Root {
				return nil, io.ErrUnexpectedEOF
			}
			return nil, nil
		}
		panic(invalidToken(t))
	}
}

func emitInstr(typ TokenType) stateFn {
	return func(p *Lexer) (stateFn, error) {
		p.instrs <- Token{typ, nil}
		return lexInstr, nil
	}
}

func lexInstrNumber(typ TokenType) stateFn {
	return func(p *Lexer) (stateFn, error) {
		arg, err := lexSigned(p)
		if err != nil {
			return nil, err
		}
		p.instrs <- Token{typ, arg}
		return lexInstr, nil
	}
}

func lexInstrLabel(typ TokenType) stateFn {
	return func(p *Lexer) (stateFn, error) {
		arg, err := lexUnsigned(p)
		if err != nil {
			return nil, err
		}
		p.instrs <- Token{typ, arg}
		return lexInstr, nil
	}
}

func lexSigned(p *Lexer) (*big.Int, error) {
	t, err := p.l.Next()
	if err != nil {
		return nil, err
	}
	switch t {
	case Space:
		return lexUnsigned(p)
	case Tab:
		num, err := lexUnsigned(p)
		if err != nil {
			return nil, err
		}
		num.Neg(num)
		return num, nil
	case LF:
		return nil, nil // zero
	case EOF:
		return nil, errors.New("unterminated number")
	}
	panic(invalidToken(t))
}

var bigOne = new(big.Int).SetInt64(1)

func lexUnsigned(p *Lexer) (*big.Int, error) {
	num := new(big.Int)
	for {
		t, err := p.l.Next()
		if err != nil {
			return nil, err
		}
		switch t {
		case Space:
			num.Lsh(num, 1)
		case Tab:
			num.Lsh(num, 1).Or(num, bigOne)
		case LF:
			return num, nil
		case EOF:
			return nil, fmt.Errorf("unterminated number: %d", num)
		default:
			panic(invalidToken(t))
		}
	}
}

func invalidToken(t SpaceToken) string {
	return fmt.Sprintf("invalid token: %d", t)
}

func init() {
	lexInstr = transition(states{
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

var lexInstr stateFn

var lexStack = transition(states{
	Space: lexInstrNumber(Push),
	Tab: transition(states{
		Space: lexInstrNumber(Copy),
		LF:    lexInstrNumber(Slide),
	}),
	LF: transition(states{
		Space: emitInstr(Dup),
		Tab:   emitInstr(Swap),
		LF:    emitInstr(Drop),
	}),
})

var lexArith = transition(states{
	Space: transition(states{
		Space: emitInstr(Add),
		Tab:   emitInstr(Sub),
		LF:    emitInstr(Mul),
	}),
	Tab: transition(states{
		Space: emitInstr(Div),
		Tab:   emitInstr(Mod),
	}),
})

var lexHeap = transition(states{
	Space: emitInstr(Store),
	Tab:   emitInstr(Retrieve),
})

var lexIO = transition(states{
	Space: transition(states{
		Space: emitInstr(Printc),
		Tab:   emitInstr(Printi),
	}),
	Tab: transition(states{
		Space: emitInstr(Readc),
		Tab:   emitInstr(Readi),
	}),
})

var lexFlow = transition(states{
	Space: transition(states{
		Space: lexInstrLabel(Label),
		Tab:   lexInstrLabel(Call),
		LF:    lexInstrLabel(Jmp),
	}),
	Tab: transition(states{
		Space: lexInstrLabel(Jz),
		Tab:   lexInstrLabel(Jn),
		LF:    emitInstr(Ret),
	}),
	LF: transition(states{
		LF: emitInstr(End),
	}),
})
