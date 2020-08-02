package ws // import "github.com/andrewarchi/nebula/ws"

import (
	"fmt"
	"go/token"
	"io"
	"math/big"
)

// Lexer is a lexical analyzer for Whitespace source.
type Lexer struct {
	file        *token.File
	src         []byte
	offset      int
	startOffset int
	endOffset   int
}

// SyntaxError identifies the location of a syntactic error.
type SyntaxError struct { // TODO report instruction string
	Msg   string
	Start token.Position
	End   token.Position
}

type stateFn func(*Lexer) (*Token, error)

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

// NewLexer constructs a Lexer scan a Whitespace source
// file into tokens.
func NewLexer(file *token.File, src []byte) *Lexer {
	return &Lexer{
		file:        file,
		src:         src,
		offset:      0,
		startOffset: 0,
		endOffset:   0,
	}
}

// Lex scans a single Whitespace token.
func (l *Lexer) Lex() (*Token, error) {
	return lexInst(l)
}

// LexProgram scans a Whitespace source file into a Program.
func (l *Lexer) LexProgram() (*Program, error) {
	var tokens []Token
	for {
		tok, err := l.Lex()
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			return &Program{l.file, tokens, nil}, nil
		}
		tokens = append(tokens, *tok)
	}
}

// Position returns the full position information for a given pos.
func (l *Lexer) Position(pos token.Pos) token.Position {
	return l.file.PositionFor(pos, false)
}

func (l *Lexer) next() (byte, bool) {
	if l.offset < len(l.src) {
		l.endOffset = l.offset
		l.offset++
		c := l.src[l.endOffset]
		if c == '\n' {
			l.file.AddLine(l.offset)
		}
		return c, false
	}
	return 0, true
}

func (l *Lexer) peek() (byte, bool) {
	if l.offset < len(l.src) {
		return l.src[l.offset], false
	}
	return 0, true
}

func (l *Lexer) emitToken(typ Type, arg *big.Int) (*Token, error) {
	return &Token{
		Type:  typ,
		Arg:   arg,
		Start: l.file.Pos(l.startOffset),
		End:   l.file.Pos(l.endOffset),
	}, nil
}

func (l *Lexer) error(msg string) (*Token, error) {
	return nil, &SyntaxError{
		Msg:   msg,
		Start: l.Position(l.file.Pos(l.startOffset)),
		End:   l.Position(l.file.Pos(l.endOffset)),
	}
}

func (l *Lexer) errorf(format string, args ...interface{}) (*Token, error) {
	return l.error(fmt.Sprintf(format, args...))
}

func transition(s states) stateFn {
	return func(l *Lexer) (*Token, error) {
	next:
		c, eof := l.next()
		if eof {
			if s.Root {
				return nil, io.EOF
			}
			return l.error("incomplete instruction")
		}
		if s.Root {
			l.startOffset = l.endOffset
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
			goto next
		}
		if state == nil {
			return l.error("invalid instruction")
		}
		return state(l)
	}
}

var (
	bigZero = big.NewInt(0)
	bigOne  = big.NewInt(1)
)

func lexNumber(typ Type, signed bool) stateFn {
	return func(l *Lexer) (*Token, error) {
		var negative bool
		if signed {
		next:
			tok, eof := l.next()
			if eof {
				return l.error("unterminated number")
			}
			switch tok {
			case space:
			case tab:
				negative = true
			case lf:
				return l.emitToken(typ, bigZero)
			default:
				goto next
			}
		}
		num := new(big.Int)
		for {
			tok, eof := l.next()
			if eof {
				return l.errorf("unterminated number: %d", num)
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
				return l.emitToken(typ, num)
			}
		}
	}
}

func emitInst(typ Type) stateFn {
	return func(l *Lexer) (*Token, error) {
		return l.emitToken(typ, nil)
	}
}

func (err *SyntaxError) Error() string {
	return fmt.Sprintf("syntax error: %s at %v - %v", err.Msg, err.Start, err.End)
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
