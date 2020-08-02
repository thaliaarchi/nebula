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

// Position returns the full position information for a given pos.
func (p *Program) Position(pos token.Pos) token.Position {
	return p.File.PositionFor(pos, false)
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

// DumpPos formats a program as Whitespace assembly with source position
// information.
func (p *Program) DumpPos() string {
	const indent = "    "
	const width = 39
	padding := strings.Repeat(" ", width)

	var b strings.Builder
	for _, tok := range p.Tokens {
		indentLen := 0
		if tok.Type != Label {
			b.WriteString(indent)
			indentLen = len(indent)
		}
		t := tok.Format(p.LabelNames)
		b.WriteString(t)
		if indentLen+len(t) < width {
			b.WriteString(padding[:width-len(t)-indentLen])
		}
		b.WriteString(" ; ")
		pos := p.Position(tok.Start)
		pos.Filename = ""
		b.WriteString(pos.String())
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
