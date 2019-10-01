package ssa

import (
	"math/big"

	"github.com/andrewarchi/wspace/token"
)

type Node interface{}

type BasicBlock []Node

type PhiExpr struct {
	Exprs []Node
}

type Literal struct {
	Value *big.Int
}

type BinaryExpr struct {
	Type token.Type
	LHS  Node
	RHS  Node
}

type RetrieveExpr struct {
	Address Node
}

type StoreStmt struct {
	Address Node
	Value   Node
}

type IOStmt struct {
	Type token.Type
	Arg  Node
}

type JmpStmt struct {
	Type    token.Type
	Address Node
}

type JmpCondStmt struct {
	Type  token.Type
	True  Node
	False Node
}

type RetStmt struct{}

type EndStmt struct{}
