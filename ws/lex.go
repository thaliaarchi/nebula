package ws

import (
	"fmt"
	"go/token"
	"io"
	"math/big"
)

// lexer is a lexical analyzer that scans tokens in Whitespace source.
type lexer struct {
	file        *token.File
	src         []byte
	tokens      []*Token
	offset      int
	startOffset int
}

// SyntaxError identifies the location of a syntactic error.
type SyntaxError struct { // TODO report instruction string
	Err string
	Pos token.Position
	End token.Position // inclusive
}

const (
	space = ' '
	tab   = '\t'
	lf    = '\n'
)

// LexTokens scans a Whitespace source file into tokens.
func LexTokens(file *token.File, src []byte) ([]*Token, error) {
	l := &lexer{file: file, src: src}
	s := rootState
	var err error
	for {
		s, err = s.nextState(l)
		if err == io.EOF {
			return l.tokens, nil
		}
		if err != nil {
			return nil, err
		}
	}
}

func (l *lexer) next() (byte, bool) {
	if l.offset < len(l.src) {
		c := l.src[l.offset]
		l.offset++
		if c == '\n' {
			l.file.AddLine(l.offset)
		}
		return c, false
	}
	return 0, true
}

func (l *lexer) error(err string) error {
	return &SyntaxError{
		Err: err,
		Pos: l.file.Position(l.file.Pos(l.startOffset)),
		End: l.file.Position(l.file.Pos(l.offset - 1)),
	}
}

func (l *lexer) errorf(format string, args ...interface{}) error {
	return l.error(fmt.Sprintf(format, args...))
}

func (err *SyntaxError) Error() string {
	end := err.End
	if err.Pos.Filename == end.Filename {
		end.Filename = ""
	}
	return fmt.Sprintf("syntax error: %s at %v-%v", err.Err, err.Pos, end)
}

type state interface {
	nextState(*lexer) (state, error)
}

type transition struct {
	Space  state
	Tab    state
	LF     state
	CanEOF bool
}

func (t *transition) nextState(l *lexer) (state, error) {
	for {
		c, eof := l.next()
		if eof {
			if t.CanEOF {
				return nil, io.EOF
			}
			return nil, l.error("incomplete instruction")
		}
		var next state
		switch c {
		case space:
			next = t.Space
		case tab:
			next = t.Tab
		case lf:
			next = t.LF
		default:
			continue
		}
		if next == nil {
			return nil, l.error("invalid instruction")
		}
		return next, nil
	}
}

type argType uint8

const (
	noArg argType = iota
	signedArg
	labelArg
)

type accept struct {
	Type Type
	Arg  argType
}

func (acc *accept) nextState(l *lexer) (state, error) {
	tok := &Token{Type: acc.Type}
	if acc.Arg != noArg {
		num, err := l.lexNumber(acc.Type, acc.Arg == signedArg)
		if err != nil {
			return nil, err
		}
		tok.Arg = num
	}
	tok.Pos = l.file.Pos(l.startOffset)
	tok.End = l.file.Pos(l.offset)
	l.startOffset = l.offset
	l.tokens = append(l.tokens, tok)
	return rootState, nil
}

var (
	bigZero = big.NewInt(0)
	bigOne  = big.NewInt(1)
)

func (l *lexer) lexNumber(typ Type, signed bool) (*big.Int, error) {
	var negative bool
	if signed {
		for {
			tok, eof := l.next()
			if eof {
				return nil, l.errorf("unterminated number: %v", typ)
			}
			switch tok {
			case space:
			case tab:
				negative = true
			case lf:
				return bigZero, nil
			default:
				continue
			}
			break
		}
	}

	num := new(big.Int)
	for {
		tok, eof := l.next()
		if eof {
			return nil, l.errorf("unterminated number: %v %d", typ, num)
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
			return num, nil
		}
	}
}

var rootState state = &transition{
	// Stack
	Space: &transition{
		Space: &accept{Push, signedArg},
		Tab: &transition{
			Space: &accept{Copy, signedArg},
			Tab: &transition{
				Space: &accept{Shuffle, noArg},
			},
			LF: &accept{Slide, signedArg},
		},
		LF: &transition{
			Space: &accept{Dup, noArg},
			Tab:   &accept{Swap, noArg},
			LF:    &accept{Drop, noArg},
		},
	},

	Tab: &transition{
		// Arithmetic
		Space: &transition{
			Space: &transition{
				Space: &accept{Add, noArg},
				Tab:   &accept{Sub, noArg},
				LF:    &accept{Mul, noArg},
			},
			Tab: &transition{
				Space: &accept{Div, noArg},
				Tab:   &accept{Mod, noArg},
			},
		},

		// Heap
		Tab: &transition{
			Space: &accept{Store, noArg},
			Tab:   &accept{Retrieve, noArg},
		},

		// I/O
		LF: &transition{
			Space: &transition{
				Space: &accept{Printc, noArg},
				Tab:   &accept{Printi, noArg},
			},
			Tab: &transition{
				Space: &accept{Readc, noArg},
				Tab:   &accept{Readi, noArg},
			},
		},
	},

	// Control flow
	LF: &transition{
		Space: &transition{
			Space: &accept{Label, labelArg},
			Tab:   &accept{Call, labelArg},
			LF:    &accept{Jmp, labelArg},
		},
		Tab: &transition{
			Space: &accept{Jz, labelArg},
			Tab:   &accept{Jn, labelArg},
			LF:    &accept{Ret, noArg},
		},
		LF: &transition{
			LF: &accept{End, noArg},

			// Debug
			Space: &transition{
				Space: &transition{
					Space: &accept{DumpStack, noArg},
					Tab:   &accept{DumpHeap, noArg},
				},
			},
			Tab: &accept{Trace, noArg},
		},

		CanEOF: true, // allow trailing LF
	},

	CanEOF: true,
}
