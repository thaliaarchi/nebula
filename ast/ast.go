package ast

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/andrewarchi/wspace/bigint"
	"github.com/andrewarchi/wspace/token"
)

// BasicBlock is a list of consecutive non-branching instructions in a
// program followed by a branch.
type BasicBlock struct {
	Labels  []*big.Int
	Tokens  []token.Token
	Flow    token.Token
	Branch  *BasicBlock
	Next    *BasicBlock
	Callers []*BasicBlock
}

func NewAST(tokens []token.Token) (*BasicBlock, error) {
	if needsImplicitEnd(tokens) {
		tokens = append(tokens, token.Token{Type: token.End})
	}
	blocks, labels, err := getBlocks(tokens)
	if err != nil {
		return nil, err
	}
	if err := annotateBlockCalls(blocks, labels); err != nil {
		return nil, err
	}
	return blocks[0], nil
}

func needsImplicitEnd(tokens []token.Token) bool {
	if len(tokens) == 0 {
		return true
	}
	switch tokens[len(tokens)-1].Type {
	case token.Call, token.Jmp, token.Ret, token.End:
		return false
	}
	return true
}

func getBlocks(tokens []token.Token) ([]*BasicBlock, *bigint.Map, error) {
	labels := bigint.NewMap(nil) // map[*big.Int]int
	var blocks []*BasicBlock
	for i := 0; i < len(tokens); i++ {
		block := &BasicBlock{}
		for tokens[i].Type == token.Label {
			label := tokens[i].Arg
			if labels.Put(label, len(blocks)) {
				return nil, nil, fmt.Errorf("ast: label is not unique: %s", label)
			}
			block.Labels = append(block.Labels, label)
			i++
		}
		for j := i; j < len(tokens); j++ {
			if tokens[j].Type.IsFlow() {
				block.Tokens = tokens[i:j]
				if tokens[j].Type == token.Label {
					block.Flow = token.Token{Type: token.Illegal}
					j--
				} else {
					block.Flow = tokens[j]
				}
				blocks = append(blocks, block)
				i = j
				break
			}
		}
	}
	return blocks, labels, nil
}

func annotateBlockCalls(blocks []*BasicBlock, labels *bigint.Map) error {
	for i, block := range blocks {
		switch block.Flow.Type {
		case token.Illegal: // implicit fallthrough jump
			if i < len(blocks)-1 {
				block.Branch = blocks[i+1]
				block.Flow.Type = token.Jmp
				blocks[i+1].Callers = append(blocks[i+1].Callers, block)
			} else {
				panic(fmt.Sprintf("ast: unexpected illegal instruction"))
			}
		case token.Call, token.Jmp, token.Jz, token.Jn:
			label, ok := labels.Get(block.Flow.Arg)
			if !ok {
				return fmt.Errorf("ast: label does not exist: %s", block.Flow.Arg)
			}
			callee := blocks[label.(int)]
			callee.Callers = append(callee.Callers, block)
			block.Branch = callee
			if i < len(blocks)-1 {
				block.Next = blocks[i+1]
			}
		}
	}
	return nil
}

func (block *BasicBlock) Display() string {
	var b strings.Builder
	for _, label := range block.Labels {
		b.WriteString("label_")
		b.WriteString(label.String())
		b.WriteString(":\n")
	}
	for _, tok := range block.Tokens {
		b.WriteString("    ")
		b.WriteString(tok.String())
		b.WriteByte('\n')
	}
	b.WriteString("    ")
	b.WriteString(block.Flow.String())
	return b.String()
}
