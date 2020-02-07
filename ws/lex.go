package ws // import "github.com/andrewarchi/nebula/ws"

import (
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"

	"github.com/andrewarchi/nebula/bigint"
)

type lexer struct {
	r      SpaceReader
	tokens []Token
	pos    Pos
}

// Lex lexically analyzes a Whitespace source to produce tokens.
func Lex(r SpaceReader) ([]Token, error) {
	l := &lexer{r: r}
	var err error
	for state := lexInst; state != nil; {
		state, err = state(l)
		if err != nil {
			return nil, err
		}
	}
	return l.tokens, nil
}

func LexProgram(filename string, bitPacked bool) (*Program, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var r SpaceReader
	if bitPacked {
		r = NewBitReader(f, filename)
	} else {
		r = NewTextReader(f, filename)
	}
	tokens, err := Lex(r)
	if err != nil {
		return nil, err
	}

	var labelNames *bigint.Map
	if info, err := os.Stat(filename + ".map"); err == nil && !info.IsDir() {
		sourceMap, err := os.Open(filename + ".map")
		if err != nil {
			return nil, err
		}
		defer sourceMap.Close()
		labelNames, err = ParseSourceMap(sourceMap)
		if err != nil {
			return nil, err
		}
	}

	return &Program{
		Name:       filename,
		Tokens:     tokens,
		LabelNames: labelNames,
	}, nil
}

func (l *lexer) appendToken(typ Type, arg *big.Int, argPos Pos) {
	l.tokens = append(l.tokens, Token{typ, arg, l.pos, argPos, l.r.Pos()})
}

type stateFn func(*lexer) (stateFn, error)

type states struct {
	Space stateFn
	Tab   stateFn
	LF    stateFn
}

func root(s states) stateFn {
	return func(l *lexer) (stateFn, error) {
		l.pos = l.r.Pos()
		return l.nextState(s, true)
	}
}

func transition(s states) stateFn {
	return func(l *lexer) (stateFn, error) {
		return l.nextState(s, false)
	}
}

func (l *lexer) nextState(s states, isRoot bool) (stateFn, error) {
	tok, err := l.r.Next()
	if err != nil {
		return nil, err
	}
	switch tok {
	case Space:
		return s.Space, nil
	case Tab:
		return s.Tab, nil
	case LF:
		return s.LF, nil
	case EOF:
		if isRoot {
			return nil, nil
		}
		return nil, io.ErrUnexpectedEOF
	default:
		panic("ws: unrecognized token")
	}
}

func emitInst(typ Type) stateFn {
	return func(l *lexer) (stateFn, error) {
		l.appendToken(typ, nil, Pos{})
		return lexInst, nil
	}
}

func lexNumber(typ Type) stateFn {
	return func(l *lexer) (stateFn, error) {
		argPos := l.r.Pos()
		arg, err := l.lexSigned()
		if err != nil {
			return nil, err
		}
		l.appendToken(typ, arg, argPos)
		return lexInst, nil
	}
}

func lexLabel(typ Type) stateFn {
	return func(l *lexer) (stateFn, error) {
		argPos := l.r.Pos()
		arg, err := l.lexUnsigned()
		if err != nil {
			return nil, err
		}
		l.appendToken(typ, arg, argPos)
		return lexInst, nil
	}
}

var (
	bigZero = big.NewInt(0)
	bigOne  = big.NewInt(1)
)

func (l *lexer) lexSigned() (*big.Int, error) {
	tok, err := l.r.Next()
	if err != nil {
		return nil, err
	}
	switch tok {
	case Space:
		return l.lexUnsigned()
	case Tab:
		num, err := l.lexUnsigned()
		if err != nil {
			return nil, err
		}
		num.Neg(num)
		return num, nil
	case LF:
		return bigZero, nil
	case EOF:
		return nil, errors.New("unterminated number")
	default:
		panic("ws: unrecognized token")
	}
}

func (l *lexer) lexUnsigned() (*big.Int, error) {
	num := new(big.Int)
	for {
		tok, err := l.r.Next()
		if err != nil {
			return nil, err
		}
		switch tok {
		case Space:
			num.Lsh(num, 1)
		case Tab:
			num.Lsh(num, 1).Or(num, bigOne)
		case LF:
			return num, nil
		case EOF:
			return nil, fmt.Errorf("unterminated number: %d", num)
		default:
			panic("ws: unrecognized token")
		}
	}
}

func init() {
	lexInst = root(states{
		Space: lexStack,
		Tab: transition(states{
			Space: lexArith,
			Tab:   lexHeap,
			LF:    lexIO,
		}),
		LF: lexFlow,
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
