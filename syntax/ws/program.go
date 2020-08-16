package ws

import (
	"bytes"
	"go/token"
	"regexp"
	"strings"

	"github.com/andrewarchi/nebula/internal/bigint"
)

// Program is a sequence of tokens with source map information.
type Program struct {
	File       *token.File
	Tokens     []*Token
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
		if tok.Type == Label {
			b.WriteString(tok.String())
			b.WriteByte(':')
		} else {
			b.WriteString(indent)
			b.WriteString(tok.String())
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// DumpPos formats a program as Whitespace assembly with source position
// information.
func (p *Program) DumpPos() string {
	const indent = "    "
	const padWidth = 39
	padding := strings.Repeat(" ", padWidth)

	var b strings.Builder
	for _, tok := range p.Tokens {
		t := tok.String()
		l := len(t)
		if tok.Type == Label {
			b.WriteString(t)
			b.WriteByte(':')
			l++
		} else {
			b.WriteString(indent)
			b.WriteString(t)
			l += len(indent)
		}
		if l < padWidth {
			b.WriteString(padding[:padWidth-l])
		}
		b.WriteString(" ; ")
		pos := p.Position(tok.Start)
		pos.Filename = ""
		b.WriteString(pos.String())
		b.WriteByte('\n')
	}
	return b.String()
}

var spacePattern = regexp.MustCompile("[ \t\n]+")

func (p *Program) DumpCommented(src []byte, indent string) string {
	var b strings.Builder
	start := 0
	for i, tok := range p.Tokens {
		end := p.File.Offset(tok.End)
		if i == len(p.Tokens)-1 {
			end = p.File.Size()
		}
		comment := bytes.TrimSpace(spacePattern.ReplaceAll(src[start:end], []byte{' '}))
		if len(comment) != 0 {
			if tok.Type != Label {
				b.WriteString(indent)
			}
			b.WriteString("; ")
			b.Write(comment)
			b.WriteByte('\n')
		}
		if tok.Type == Label {
			b.WriteString(tok.String())
			b.WriteByte(':')
		} else {
			b.WriteString(indent)
			b.WriteString(tok.String())
		}
		b.WriteByte('\n')
		start = end
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
