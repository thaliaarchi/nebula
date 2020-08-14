// tetrisrun is a runner for Peter De Wachter's Whitespace Tetris game.
//
// It moves blocks downwards automatically on the interval specified
// with the -speed flag and maps arrow keys to i/j/k/ls.
//
// Download tetris.ws from the Whitespace mailing list archives:
// https://web.archive.org/web/20141011193149/http://compsoc.dur.ac.uk/archives/whitespace/2008-January/000067.html
//
// Running:
//
//     ./compile tetris.ws build/tetris
//     stty raw -echo && go run programs/tetrisrun.go -speed 750ms | build/tetris
//
// Controls:
//
//     j, left arrow - move left
//     k, down arrow - drop
//     l, right arrow - move right
//     i, up arrow - rotate
//
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var stdin = bufio.NewReader(os.Stdin)

func main() {
	speed := flag.Duration("speed", 750*time.Millisecond, "drop rate")
	flag.Parse()

	ticker := time.NewTicker(*speed)
	signal.Ignore(syscall.SIGPIPE)
	done := make(chan bool)

	go func() {
		for {
			select {
			default:
				key, err := readKey()
				try(err)
				_, err = os.Stdout.Write([]byte{key})
				if isSIGPIPE(err) || err == io.EOF {
					done <- true
					return
				}
				try(err)
			case <-done:
				return
			}
		}
	}()

Loop:
	for {
		select {
		case <-ticker.C:
			_, err := os.Stdout.WriteString("k")
			if isSIGPIPE(err) {
				done <- true
				break Loop
			}
			try(err)
		case <-done:
			break Loop
		}
	}

	ticker.Stop()
}

// readKey translates arrow keys to i, j, j, and l and filters out
// other keys.
func readKey() (byte, error) {
	var prev byte
	escape := false
	for {
		b, err := stdin.ReadByte()
		if err != nil {
			return 0, err
		}
		switch {
		case 'i' <= b && b <= 'l':
			return b, nil
		case prev == '\x1b' && b == '[': // ESC [
			escape = true
			prev = b
		case b == 'A' && escape: // up arrow
			return 'i', nil
		case b == 'B' && escape: // down arrow
			return 'k', nil
		case b == 'C' && escape: // right arrow
			return 'l', nil
		case b == 'D' && escape: // left arrow
			return 'j', nil
		default:
			escape = false
			prev = b
		}
	}
}

func isSIGPIPE(err error) bool {
	patherr, ok := err.(*os.PathError)
	return ok && patherr.Err == syscall.EPIPE
}

func try(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
