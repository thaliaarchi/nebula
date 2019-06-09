package ws

import (
	"errors"
	"fmt"
	"io"
	"math/big"
)

type Parser struct {
	l      SpaceLexer
	instrs chan Instr
}

func Parse(l SpaceLexer) chan Instr {
	p := &Parser{
		l:      l,
		instrs: make(chan Instr),
	}
	go p.run()
	return p.instrs
}

func (p *Parser) run() error {
	defer close(p.instrs)
	var err error
	for state := parseInstr; state != nil; {
		state, err = state(p)
		if err != nil {
			return err
		}
	}
	return nil
}

type stateFn func(*Parser) (stateFn, error)

type states struct {
	Space stateFn
	Tab   stateFn
	LF    stateFn
	Root  bool
}

func transition(s states) stateFn {
	return func(p *Parser) (stateFn, error) {
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

func emitInstr(typ InstrType) stateFn {
	return func(p *Parser) (stateFn, error) {
		p.instrs <- Instr{typ, nil}
		return parseInstr, nil
	}
}

func parseInstrNumber(typ InstrType) stateFn {
	return func(p *Parser) (stateFn, error) {
		arg, err := parseSigned(p)
		if err != nil {
			return nil, err
		}
		p.instrs <- Instr{typ, arg}
		return parseInstr, nil
	}
}

func parseInstrLabel(typ InstrType) stateFn {
	return func(p *Parser) (stateFn, error) {
		arg, err := parseUnsigned(p)
		if err != nil {
			return nil, err
		}
		p.instrs <- Instr{typ, arg}
		return parseInstr, nil
	}
}

func parseSigned(p *Parser) (*big.Int, error) {
	t, err := p.l.Next()
	if err != nil {
		return nil, err
	}
	switch t {
	case Space:
		return parseUnsigned(p)
	case Tab:
		num, err := parseUnsigned(p)
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

func parseUnsigned(p *Parser) (*big.Int, error) {
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

func invalidToken(t Token) string {
	return fmt.Sprintf("invalid token: %d", t)
}

func init() {
	parseInstr = transition(states{
		Space: parseStack,
		Tab: transition(states{
			Space: parseArith,
			Tab:   parseHeap,
			LF:    parseIO,
		}),
		LF:   parseFlow,
		Root: true,
	})
}

var parseInstr stateFn

var parseStack = transition(states{
	Space: parseInstrNumber(Push),
	Tab: transition(states{
		Space: parseInstrNumber(Copy),
		LF:    parseInstrNumber(Slide),
	}),
	LF: transition(states{
		Space: emitInstr(Dup),
		Tab:   emitInstr(Swap),
		LF:    emitInstr(Drop),
	}),
})

var parseArith = transition(states{
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

var parseHeap = transition(states{
	Space: emitInstr(Store),
	Tab:   emitInstr(Retrieve),
})

var parseIO = transition(states{
	Space: transition(states{
		Space: emitInstr(Printc),
		Tab:   emitInstr(Printi),
	}),
	Tab: transition(states{
		Space: emitInstr(Readc),
		Tab:   emitInstr(Readi),
	}),
})

var parseFlow = transition(states{
	Space: transition(states{
		Space: parseInstrLabel(Label),
		Tab:   parseInstrLabel(Call),
		LF:    parseInstrLabel(Jmp),
	}),
	Tab: transition(states{
		Space: parseInstrLabel(Jz),
		Tab:   parseInstrLabel(Jn),
		LF:    emitInstr(Ret),
	}),
	LF: transition(states{
		LF: emitInstr(End),
	}),
})
