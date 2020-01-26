package ws // import "github.com/andrewarchi/nebula/ws"

import (
	"errors"
	"fmt"
	"io"
	"math/big"
)

type lexer struct {
	r      SpaceReader
	tokens []Token
}

// Lex lexically analyzes a Whitespace source to produce tokens.
func Lex(r SpaceReader) ([]Token, error) {
	l := &lexer{r: r}
	var err error
	for state := lexInstr; state != nil; {
		state, err = state(l)
		if err != nil {
			return nil, err
		}
	}
	return l.tokens, nil
}

func (l *lexer) appendToken(tok Token) {
	l.tokens = append(l.tokens, tok)
}

type stateFn func(*lexer) (stateFn, error)

type states struct {
	Space stateFn
	Tab   stateFn
	LF    stateFn
	Root  bool
}

func transition(s states) stateFn {
	return func(p *lexer) (stateFn, error) {
		tok, err := p.r.Next()
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
			if !s.Root {
				return nil, io.ErrUnexpectedEOF
			}
			return nil, nil
		}
		panic(invalidToken(tok))
	}
}

func emitInstr(typ Type) stateFn {
	return func(l *lexer) (stateFn, error) {
		l.appendToken(Token{typ, nil})
		return lexInstr, nil
	}
}

func lexInstrNumber(typ Type) stateFn {
	return func(l *lexer) (stateFn, error) {
		arg, err := lexSigned(l)
		if err != nil {
			return nil, err
		}
		l.appendToken(Token{typ, arg})
		return lexInstr, nil
	}
}

func lexInstrLabel(typ Type) stateFn {
	return func(l *lexer) (stateFn, error) {
		arg, err := lexUnsigned(l)
		if err != nil {
			return nil, err
		}
		l.appendToken(Token{typ, arg})
		return lexInstr, nil
	}
}

func lexSigned(l *lexer) (*big.Int, error) {
	tok, err := l.r.Next()
	if err != nil {
		return nil, err
	}
	switch tok {
	case Space:
		return lexUnsigned(l)
	case Tab:
		num, err := lexUnsigned(l)
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
	panic(invalidToken(tok))
}

var bigOne = big.NewInt(1)

func lexUnsigned(l *lexer) (*big.Int, error) {
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
			panic(invalidToken(tok))
		}
	}
}

func invalidToken(tok SpaceToken) string {
	return fmt.Sprintf("invalid token: %d", tok)
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
