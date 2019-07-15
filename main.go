package main

import (
	"fmt"
	"os"

	"github.com/andrewarchi/wspace/ws"
)

const usage = `wspace run [file]
wspace debug [file]`

func main() {
	if len(os.Args) != 3 {
		fmt.Println(usage)
		return
	}
	mode := os.Args[1]
	if mode != "run" && mode != "debug" {
		fmt.Println("unrecognized mode")
		fmt.Println(usage)
		return
	}
	f, err := os.Open(os.Args[2])
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
	if mode == "run" {
		vm.Run()
	} else {
		vm.Debug()
	}
}
