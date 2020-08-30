// Copyright 2018 The Go Authors. All rights reserved.
// Copyright 2020 Andrew Archibald. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syntax

import "fmt"

// PosMax is the largest line or column value that can be represented without loss.
// Incoming values (arguments) larger than PosMax will be set to PosMax.
const PosMax = 1 << 30

// A Pos represents an absolute (line, col) source position with. Pos
// values are intentionally light-weight so that they can be created
// without too much concern about space use.
type Pos struct {
	filename  string
	line, col uint32
}

// MakePos returns a new Pos for the given FileBase, line and column.
func MakePos(filename string, line, col uint) Pos {
	return Pos{filename, sat32(line), sat32(col)}
}

// TODO(gri) IsKnown makes an assumption about linebase < 1.
//           Maybe we should check for Base() != nil instead.

func (pos Pos) IsKnown() bool    { return pos.line > 0 }
func (pos Pos) Filename() string { return pos.filename }
func (pos Pos) Line() uint       { return uint(pos.line) }
func (pos Pos) Col() uint        { return uint(pos.col) }

func (pos Pos) String() string {
	if pos.line == 0 {
		if pos.filename == "" {
			return "<unknown position>"
		}
		return pos.filename
	}
	if pos.col == 0 {
		return fmt.Sprintf("%s:%d", pos.filename, pos.line)
	}
	return fmt.Sprintf("%s:%d:%d", pos.filename, pos.line, pos.col)
}

func sat32(x uint) uint32 {
	if x > PosMax {
		return PosMax
	}
	return uint32(x)
}
