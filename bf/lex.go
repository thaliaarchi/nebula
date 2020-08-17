package bf

import (
	"go/token"
	"io"
)

// Lexer scans tokens in Brainfuck source.
type Lexer struct {
	file   *token.File
	src    []byte
	offset int
}

// NewLexer constructs a Brainfuck lexer.
func NewLexer(file *token.File, src []byte) *Lexer {
	return &Lexer{
		file:   file,
		src:    src,
		offset: 0,
	}
}

// NextToken scans a single Brainfuck token.
func (l *Lexer) NextToken() (*Token, error) {
	for l.offset < len(l.src) {
		var typ Type
		switch l.src[l.offset] {
		case '>':
			typ = IncPtr
		case '<':
			typ = DecPtr
		case '+':
			typ = IncData
		case '-':
			typ = DecData
		case '.':
			typ = Print
		case ',':
			typ = Read
		case '[':
			typ = Bracket
		case ']':
			typ = EndBracket
		default:
			l.offset++
			continue
		}
		tok := &Token{typ, l.file.Pos(l.offset)}
		l.offset++
		return tok, nil
	}
	return nil, io.EOF
}

// LexTokens scans a Brainfuck source files into tokens.
func LexTokens(file *token.File, src []byte) ([]*Token, error) {
	l := NewLexer(file, src)
	var tokens []*Token
	for {
		tok, err := l.NextToken()
		if err == io.EOF {
			return tokens, nil
		}
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
	}
}
