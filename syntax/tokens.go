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

func (tok token) String() string {
	switch tok {
	case EOF:
		return "eof"
	case Ident:
		return "ident"
	case Int:
		return "int"
	case Float:
		return "float"
	case Rune:
		return "rune"
	case String:
		return "string"
	case Comment:
		return "comment"
	case Semi:
		return "semi"
	case Colon:
		return "colon"
	default:
		return "badtoken"
	}
}
