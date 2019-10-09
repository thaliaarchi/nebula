package ws

import (
	"bufio"
	"fmt"
	"io"
	"math/big"

	"github.com/andrewarchi/wspace/bigint"
)

// ParseSourceMap reads a source map and parses it into mappings from
// label to source name.
func ParseSourceMap(r io.Reader) (*bigint.Map, error) {
	br := bufio.NewReader(r)
	labels := bigint.NewMap(nil) // map[*big.Int]string
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
