package ws

import (
	"bufio"
	"fmt"
	"io"
	"math/big"

	"github.com/andrewarchi/nebula/internal/bigint"
)

// ParseLabelMap reads a label source map and parses it into mappings
// from label name to integer value.
func ParseLabelMap(r io.Reader) (*bigint.Map, error) {
	br := bufio.NewReader(r)
	labels := bigint.NewMap() // map[*big.Int]string
	for {
		labelText, err := br.ReadString(':')
		if err == io.EOF {
			return labels, nil
		} else if err != nil {
			return nil, err
		}
		labelText = labelText[:len(labelText)-1]
		label, ok := new(big.Int).SetString(labelText, 10)
		if !ok {
			return nil, fmt.Errorf("invalid source map label: %v", labelText)
		}
		name, err := br.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if labels.Put(label, name[:len(name)-1]) {
			return nil, fmt.Errorf("duplicate source map label: %v", labelText)
		}
	}
}

// ApplyLabelMap adds label names from mapping to tokens.
func ApplyLabelMap(tokens []*Token, labelNames *bigint.Map /* map[*big.Int]string */) {
	for _, tok := range tokens {
		if tok.Type == Label && tok.ArgString == "" {
			if name, ok := labelNames.Get(tok.Arg); ok {
				tok.ArgString = name.(string)
			}
		}
	}
}
