package ws

import "fmt"

// Pos represents a position in a source file.
type Pos struct {
	Filename string
	Line     int
	Col      int
}

func (p Pos) String() string {
	return fmt.Sprintf("%s:%d:%d", p.Filename, p.Line, p.Col)
}
