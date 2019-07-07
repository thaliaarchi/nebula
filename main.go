package main

import (
	"fmt"
	"os"

	"github.com/andrewarchi/wspace/ws"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("wspace [file]")
		return
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}
	r := ws.NewTextReader(f)
	tokenChan := ws.Lex(r)
	var tokens []ws.Token
	for token := range tokenChan {
		tokens = append(tokens, token)
	}
	vm, err := ws.NewVM(tokens)
	if err != nil {
		fmt.Println(err)
		return
	}
	vm.Run()
}
