package ws

import (
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/andrewarchi/wspace/token"
)

type Lexer struct {
	l      SpaceReader
	instrs chan token.Token
}

func Lex(l SpaceReader) <-chan token.Token {
	p := &Lexer{
		l:      l,
		instrs: make(chan token.Token),
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

func emitInstr(typ token.Type) stateFn {
	return func(p *Lexer) (stateFn, error) {
		p.instrs <- token.Token{typ, nil}
		return lexInstr, nil
	}
}

func lexInstrNumber(typ token.Type) stateFn {
	return func(p *Lexer) (stateFn, error) {
		arg, err := lexSigned(p)
		if err != nil {
			return nil, err
		}
		p.instrs <- token.Token{typ, arg}
		return lexInstr, nil
	}
}

func lexInstrLabel(typ token.Type) stateFn {
	return func(p *Lexer) (stateFn, error) {
		arg, err := lexUnsigned(p)
		if err != nil {
			return nil, err
		}
		p.instrs <- token.Token{typ, arg}
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
	Space: lexInstrNumber(token.Push),
	Tab: transition(states{
		Space: lexInstrNumber(token.Copy),
		LF:    lexInstrNumber(token.Slide),
	}),
	LF: transition(states{
		Space: emitInstr(token.Dup),
		Tab:   emitInstr(token.Swap),
		LF:    emitInstr(token.Drop),
	}),
})

var lexArith = transition(states{
	Space: transition(states{
		Space: emitInstr(token.Add),
		Tab:   emitInstr(token.Sub),
		LF:    emitInstr(token.Mul),
	}),
	Tab: transition(states{
		Space: emitInstr(token.Div),
		Tab:   emitInstr(token.Mod),
	}),
})

var lexHeap = transition(states{
	Space: emitInstr(token.Store),
	Tab:   emitInstr(token.Retrieve),
})

var lexIO = transition(states{
	Space: transition(states{
		Space: emitInstr(token.Printc),
		Tab:   emitInstr(token.Printi),
	}),
	Tab: transition(states{
		Space: emitInstr(token.Readc),
		Tab:   emitInstr(token.Readi),
	}),
})

var lexFlow = transition(states{
	Space: transition(states{
		Space: lexInstrLabel(token.Label),
		Tab:   lexInstrLabel(token.Call),
		LF:    lexInstrLabel(token.Jmp),
	}),
	Tab: transition(states{
		Space: lexInstrLabel(token.Jz),
		Tab:   lexInstrLabel(token.Jn),
		LF:    emitInstr(token.Ret),
	}),
	LF: transition(states{
		LF: emitInstr(token.End),
	}),
})
