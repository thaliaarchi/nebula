package syntax

type token uint

const (
	EOF token = iota + 1
	Ident

	// Literals
	Int
	Float
	Rune
	String
	Comment

	Semi
	Colon
)
