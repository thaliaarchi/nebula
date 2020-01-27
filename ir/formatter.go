package ir // import "github.com/andrewarchi/nebula/ir"

import (
	"fmt"
	"strings"

	"github.com/andrewarchi/nebula/bigint"
)

type formatter struct {
	IDs    map[Val]int
	NextID int
}

func newFormatter() *formatter {
	return &formatter{
		IDs:    make(map[Val]int),
		NextID: 0,
	}
}

func (f *formatter) FormatNode(node Node) string {
	switch n := node.(type) {
	case *ArithExpr:
		if n.Op == Neg {
			return fmt.Sprintf("%s = neg %s", f.FormatVal(*n.Assign), f.FormatVal(*n.LHS))
		}
		return fmt.Sprintf("%s = %v %s %s", f.FormatVal(*n.Assign), n.Op, f.FormatVal(*n.LHS), f.FormatVal(*n.RHS))
	case *LoadExpr:
		return fmt.Sprintf("%s = load *%s", f.FormatVal(*n.Assign), f.FormatVal(*n.Addr))
	case *StoreStmt:
		return fmt.Sprintf("store *%s %s", f.FormatVal(*n.Addr), f.FormatVal(*n.Val))
	case *PrintStmt:
		return fmt.Sprintf("%v %s", n.Op, f.FormatVal(*n.Val))
	case *ReadExpr:
		return fmt.Sprintf("%s = %v", f.FormatVal(*n.Assign), n.Op)
	case *CallStmt:
		return fmt.Sprintf("call %s", n.Dest.Name())
	case *JmpStmt:
		return fmt.Sprintf("%v %s", n.Op, n.Dest.Name())
	case *JmpCondStmt:
		return fmt.Sprintf("%v %s %s %s", n.Op, f.FormatVal(*n.Cond), n.Then.Name(), n.Else.Name())
	case *RetStmt:
		return "ret"
	case *ExitStmt:
		return "exit"
	}
	panic("ir: unrecognized node type")
}

func (f *formatter) FormatVal(val Val) string {
	switch v := val.(type) {
	case *StackVal:
		id := v.ID
		if id > 0 {
			if vid, ok := f.IDs[val]; ok {
				id = vid
			} else {
				id = f.NextID
				f.IDs[val] = f.NextID
				f.NextID++
			}
		}
		return fmt.Sprintf("%%%d", id)
	case *ConstVal:
		return v.Int.String()
	case *StringVal:
		return fmt.Sprintf("%q", v.Str)
	case *ArrayVal:
		return bigint.FormatSlice(v.Array)
	case *PhiVal:
		return fmt.Sprintf("phi(%s)", f.FormatSlice(v.Vals))
	}
	panic("ir: unrecognized val type")
}

func (f *formatter) FormatSlice(vals []*Val) string {
	var b strings.Builder
	for i, val := range vals {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(f.FormatVal(*val))
	}
	return b.String()
}
