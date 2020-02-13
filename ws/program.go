package ws // import "github.com/andrewarchi/nebula/ws"

import (
	"go/token"
	"strings"

	"github.com/andrewarchi/nebula/bigint"
)

// Program is a sequence of tokens with source map information.
type Program struct {
	File       *token.File
	Tokens     []Token
	LabelNames *bigint.Map // map[*big.Int]string
}

// Dump formats a program as Whitespace assembly.
func (p *Program) Dump(indent string) string {
	var b strings.Builder
	for _, tok := range p.Tokens {
		if tok.Type != Label {
			b.WriteString(indent)
		}
		b.WriteString(tok.Format(p.LabelNames))
		b.WriteByte('\n')
	}
	return b.String()
}

// DumpWS formats a program as Whitespace.
func (p *Program) DumpWS() string {
	var b strings.Builder
	for _, tok := range p.Tokens {
		b.WriteString(tok.StringWS())
	}
	return b.String()
}

func (p *Program) String() string {
	return p.Dump("    ")
}
