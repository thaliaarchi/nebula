package wspace

import (
	"fmt"
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

func treeFn(spaceFn, tabFn, lfFn stateFn) stateFn {
	return func(p *Parser) (stateFn, error) {
		t, err := p.l.Next()
		if err != nil {
			return nil, err
		}
		switch t {
		case Space:
			return spaceFn, nil
		case Tab:
			return tabFn, nil
		case LF:
			return lfFn, nil
		case EOF:
			return nil, nil // TODO
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
			num.Lsh(num, 1)
			num.Add(num, bigOne)
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
	parseInstr = treeFn(
		parseStack,
		treeFn(parseArith, parseHeap, parseIO),
		parseFlow,
	)
}

var parseInstr stateFn

var parseStack = treeFn(
	instrNumberFn(Push),
	treeFn(instrNumberFn(Copy), nil, instrNumberFn(Slide)),
	treeFn(instrFn(Dup), instrFn(Swap), instrFn(Drop)),
)

var parseArith = treeFn(
	treeFn(instrFn(Add), instrFn(Sub), instrFn(Mul)),
	treeFn(instrFn(Div), instrFn(Mod), nil),
	nil,
)

var parseHeap = treeFn(
	instrFn(Store),
	instrFn(Retrieve),
	nil,
)

var parseIO = treeFn(
	treeFn(instrFn(Printc), instrFn(Printi), nil),
	treeFn(instrFn(Readc), instrFn(Readi), nil),
	nil,
)

var parseFlow = treeFn(
	treeFn(instrLabelFn(Label), instrLabelFn(Call), instrLabelFn(Jmp)),
	treeFn(instrLabelFn(Jz), instrLabelFn(Jn), instrFn(Ret)),
	treeFn(nil, nil, instrFn(End)),
)
