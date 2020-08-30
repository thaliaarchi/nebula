package wsa

import "github.com/andrewarchi/nebula/ws"

var instNames = map[string]ws.Type{
	"push":      ws.Push,
	"dup":       ws.Dup,
	"copy":      ws.Copy,
	"swap":      ws.Swap,
	"drop":      ws.Drop,
	"slide":     ws.Slide,
	"shuffle":   ws.Shuffle,
	"add":       ws.Add,
	"sub":       ws.Sub,
	"mul":       ws.Mul,
	"div":       ws.Div,
	"mod":       ws.Mod,
	"store":     ws.Store,
	"retrieve":  ws.Retrieve,
	"label":     ws.Label,
	"call":      ws.Call,
	"jmp":       ws.Jmp,
	"jz":        ws.Jz,
	"jn":        ws.Jn,
	"ret":       ws.Ret,
	"end":       ws.End,
	"printc":    ws.Printc,
	"printi":    ws.Printi,
	"readc":     ws.Readc,
	"readi":     ws.Readi,
	"trace":     ws.Trace,
	"dumpstack": ws.DumpStack,
	"dumpheap":  ws.DumpHeap,

	// Aliases
	"duplicate":        ws.Dup,       //
	"doub":             ws.Dup,       // burghard
	"pop":              ws.Drop,      // burghard
	"load":             ws.Retrieve,  //
	"retrive":          ws.Retrieve,  // burghard
	"jump":             ws.Jmp,       // burghard
	"jumpz":            ws.Jz,        // burghard
	"jumpn":            ws.Jn,        // burghard
	"return":           ws.Ret,       //
	"exit":             ws.End,       // burghard
	"halt":             ws.End,       //
	"putc":             ws.Printc,    //
	"outc":             ws.Printc,    // burghard
	"puti":             ws.Printi,    //
	"putn":             ws.Printi,    //
	"outi":             ws.Printi,    //
	"outn":             ws.Printi,    // burghard
	"getc":             ws.Readc,     //
	"inc":              ws.Readc,     // burghard
	"geti":             ws.Readi,     //
	"getn":             ws.Readi,     //
	"ini":              ws.Readi,     //
	"inn":              ws.Readi,     // burghard
	"debug_printstack": ws.DumpStack, // burghard
	"debug_printheap":  ws.DumpHeap,  // burghard
}
