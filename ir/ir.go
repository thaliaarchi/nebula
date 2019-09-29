package ir

import "github.com/andrewarchi/wspace/token"

type BasicBlock struct {
	Nodes      []token.Token
	Next       *BasicBlock
	Branch     *BasicBlock
	BranchType BranchType
}

type BranchType uint8

const (
	Call BranchType = iota
	Jmp
	Jz
	Jn
	Ret
	End
	Fallthrough
)
