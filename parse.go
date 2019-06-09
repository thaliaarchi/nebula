package wspace

import (
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

type tree struct {
	Space stateFn
	Tab   stateFn
	LF    stateFn
	Root  bool
}

func treeFn(state tree) stateFn {
	return func(p *Parser) (stateFn, error) {
		t, err := p.l.Next()
		if err != nil {
			return nil, err
		}
		switch t {
		case Space:
			return state.Space, nil
		case Tab:
			return state.Tab, nil
		case LF:
			return state.LF, nil
		case EOF:
			if !state.Root {
				return nil, io.ErrUnexpectedEOF
			}
			return nil, nil
		}
		panic(invalidToken(t))
	}
}

func instrFn(typ InstrType) stateFn {
	return func(p *Parser) (stateFn, error) {
		p.instrs <- Instr{typ, nil}
		return parseInstr, nil
	}
}

func instrNumberFn(typ InstrType) stateFn {
	return func(p *Parser) (stateFn, error) {
		arg, err := p.parseSigned()
		if err != nil {
			return nil, err
		}
		p.instrs <- Instr{typ, arg}
		return parseInstr, nil
	}
}

func instrLabelFn(typ InstrType) stateFn {
	return func(p *Parser) (stateFn, error) {
		arg, err := p.parseUnsigned()
		if err != nil {
			return nil, err
		}
		p.instrs <- Instr{typ, arg}
		return parseInstr, nil
	}
}

func (p *Parser) parseSigned() (*big.Int, error) {
	t, err := p.l.Next()
	if err != nil {
		return nil, err
	}
	switch t {
	case Space:
		return p.parseUnsigned()
	case Tab:
		num, err := p.parseUnsigned()
		if err != nil {
			return nil, err
		}
		num.Neg(num)
		return num, nil
	case LF, EOF:
		return nil, nil // zero
	}
	panic(invalidToken(t))
}

var bigOne = new(big.Int).SetInt64(1)

func (p *Parser) parseUnsigned() (*big.Int, error) {
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
	parseInstr = treeFn(tree{
		Space: parseStack,
		Tab: treeFn(tree{
			Space: parseArith,
			Tab:   parseHeap,
			LF:    parseIO,
		}),
		LF:   parseFlow,
		Root: true,
	})
}

var parseInstr stateFn

var parseStack = treeFn(tree{
	Space: instrNumberFn(Push),
	Tab: treeFn(tree{
		Space: instrNumberFn(Copy),
		LF:    instrNumberFn(Slide),
	}),
	LF: treeFn(tree{
		Space: instrFn(Dup),
		Tab:   instrFn(Swap),
		LF:    instrFn(Drop),
	}),
})

var parseArith = treeFn(tree{
	Space: treeFn(tree{
		Space: instrFn(Add),
		Tab:   instrFn(Sub),
		LF:    instrFn(Mul),
	}),
	Tab: treeFn(tree{
		Space: instrFn(Div),
		Tab:   instrFn(Mod),
	}),
})

var parseHeap = treeFn(tree{
	Space: instrFn(Store),
	Tab:   instrFn(Retrieve),
})

var parseIO = treeFn(tree{
	Space: treeFn(tree{
		Space: instrFn(Printc),
		Tab:   instrFn(Printi),
	}),
	Tab: treeFn(tree{
		Space: instrFn(Readc),
		Tab:   instrFn(Readi),
	}),
})

var parseFlow = treeFn(tree{
	Space: treeFn(tree{
		Space: instrLabelFn(Label),
		Tab:   instrLabelFn(Call),
		LF:    instrLabelFn(Jmp),
	}),
	Tab: treeFn(tree{
		Space: instrLabelFn(Jz),
		Tab:   instrLabelFn(Jn),
		LF:    instrFn(Ret),
	}),
	LF: treeFn(tree{
		LF: instrFn(End),
	}),
})
