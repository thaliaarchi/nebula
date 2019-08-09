package ws

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/andrewarchi/wspace/bigint"
)

// Node is a node in an AST linked by program flow.
type Node struct {
	Token
	Labels  []*big.Int
	Next    *Node
	Branch  *Node
	Callers []*Node
}

func NewAST(tokens []Token) (*Node, error) {
	nodes, labels, err := getNodes(tokens)
	if err != nil {
		return nil, err
	}
	callers, callees, err := getNodeCalls(nodes, labels)
	if err != nil {
		return nil, err
	}
	annotateNodes(nodes, callers, callees)
	return nodes[0], nil
}

func getNodes(tokens []Token) ([]*Node, *bigint.Map, error) {
	nodes := make([]*Node, 0, len(tokens)+1)
	labels := bigint.NewMap(nil) // map[*big.Int]int
	var nodeLabels []*big.Int
	for _, token := range tokens {
		if token.Type == Label {
			nodeLabels = append(nodeLabels, token.Arg)
			if labels.Put(token.Arg, len(nodes)) {
				return nil, nil, fmt.Errorf("ast: label is not unique: %s", token.Arg)
			}
			continue
		}
		nodes = append(nodes, &Node{Token: token, Labels: nodeLabels})
		nodeLabels = nil
	}
	if needsImplicitEnd(nodes, nodeLabels) {
		nodes = append(nodes, &Node{Token: Token{Type: End}, Labels: nodeLabels})
	}
	return nodes, labels, nil
}

func needsImplicitEnd(nodes []*Node, endLabels []*big.Int) bool {
	if len(nodes) == 0 || len(endLabels) > 0 {
		return true
	}
	switch nodes[len(nodes)-1].Type {
	case Call, Jmp, Ret, End:
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
		case Call, Jmp, Jz, Jn:
			label, ok := labels.Get(node.Arg)
			if !ok {
				return nil, nil, fmt.Errorf("ast: label does not exist: %s", node.Arg)
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
		case Call, Jmp:
			node.Branch = callees[node]
		case Jz, Jn:
			node.Branch = callees[node]
		}
		if i < len(nodes)-1 {
			node.Next = nodes[i+1]
		}
	}
}

func (node *Node) Display() string {
	var b strings.Builder
	for _, label := range node.Labels {
		b.WriteString("label_")
		b.WriteString(label.String())
		b.WriteString(":\n")
	}
	b.WriteString("    ")
	b.WriteString(node.Token.String())
	return b.String()
}
