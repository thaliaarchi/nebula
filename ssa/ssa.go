package ssa

import (
	"math/big"

	"github.com/andrewarchi/wspace/ws"
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
	Type ws.TokenType
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
	Type ws.TokenType
	Arg  Node
}

type JmpStmt struct {
	Type    ws.TokenType
	Address Node
}

type JmpCondStmt struct {
	Type  ws.TokenType
	True  Node
	False Node
}

type RetStmt struct{}

type EndStmt struct{}
