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

func (l *lexer) appendToken(typ Type, arg *big.Int, argPos Pos) {
	l.tokens = append(l.tokens, Token{typ, arg, l.pos, argPos, l.r.Pos()})
}

type stateFn func(*lexer) (stateFn, error)

type states struct {
	Space stateFn
	Tab   stateFn
	LF    stateFn
	Root  bool
}

func transition(s states) stateFn {
	return func(l *lexer) (stateFn, error) {
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
			if !s.Root {
				return nil, io.ErrUnexpectedEOF
			}
			return nil, nil
		}
		panic(invalidToken(tok))
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
	}
	panic(invalidToken(tok))
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
			panic(invalidToken(tok))
		}
	}
}

func invalidToken(tok SpaceToken) string {
	return fmt.Sprintf("ws: invalid token: %d", tok)
}

func init() {
	lexInst = func(l *lexer) (stateFn, error) {
		l.pos = l.r.Pos()
		return transition(states{
			Space: lexStack,
			Tab: transition(states{
				Space: lexArith,
				Tab:   lexHeap,
				LF:    lexIO,
			}),
			LF:   lexFlow,
			Root: true,
		})(l)
	}
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
