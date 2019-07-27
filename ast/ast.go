package graph

import (
	"fmt"
	"math/big"

	"github.com/andrewarchi/wspace/bigint"
	"github.com/andrewarchi/wspace/ws"
)

// AST is a flow graph linking nodes by program flow.
// The first node is the program entry point.
type AST []*Node

type Node struct {
	ws.Token
	Labels   []*big.Int
	Callers  []*Node
	Branches []*Node
	Visited  bool
}

func NewAST(tokens []ws.Token) (AST, error) {
	nodes, labels, err := getNodes(tokens)
	if err != nil {
		return nil, err
	}
	callers, callees, err := getNodeCalls(nodes, labels)
	if err != nil {
		return nil, err
	}
	annotateNodes(nodes, callers, callees)
	return nodes, nil
}

func getNodes(tokens []ws.Token) ([]*Node, *bigint.Map, error) {
	nodes := make([]*Node, 0, len(tokens)+1)
	labels := bigint.NewMap(nil) // map[*big.Int]int
	var nodeLabels []*big.Int
	for _, token := range tokens {
		if token.Type == ws.Label {
			nodeLabels = append(nodeLabels, token.Arg)
			if labels.Put(token.Arg, len(nodes)) {
				return nil, nil, fmt.Errorf("graph: label is not unique: %s", token.Arg)
			}
			continue
		}
		nodes = append(nodes, &Node{Token: token, Labels: nodeLabels})
		nodeLabels = nil
	}
	if needsImplicitEnd(nodes, nodeLabels) {
		nodes = append(nodes, &Node{Token: ws.Token{Type: ws.End}, Labels: nodeLabels})
	}
	return nodes, labels, nil
}

func needsImplicitEnd(nodes []*Node, endLabels []*big.Int) bool {
	if len(nodes) == 0 || len(endLabels) != 0 {
		return true
	}
	switch nodes[len(nodes)-1].Type {
	case ws.Call, ws.Jmp, ws.Ret, ws.End:
	default:
		return true
	}
	return false
}

func getNodeCalls(nodes []*Node, labels *bigint.Map) (map[*Node][]*Node, map[*Node]*Node, error) {
	callers := make(map[*Node][]*Node)
	callees := make(map[*Node]*Node)
	for _, node := range nodes {
		switch node.Type {
		case ws.Call, ws.Jmp, ws.Jz, ws.Jn:
			label, ok := labels.Get(node.Arg)
			if !ok {
				return nil, nil, fmt.Errorf("graph: label does not exist: %s", node.Arg)
			}
			callee := nodes[label.(int)]
			callers[callee] = append(callers[callee], node)
			callees[node] = callee
		}
	}
	return callers, callees, nil
}

func annotateNodes(nodes []*Node, callers map[*Node][]*Node, callees map[*Node]*Node) {
	for i, node := range nodes {
		node.Callers = callers[node]
		switch node.Type {
		case ws.Call, ws.Jmp:
			node.Branches = []*Node{callees[node]}
		case ws.Jz, ws.Jn:
			node.Branches = []*Node{callees[node], nodes[i+1]}
		case ws.Ret, ws.End:
		default:
			node.Branches = []*Node{nodes[i+1]}
		}
	}
}
