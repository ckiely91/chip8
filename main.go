package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"time"

	termbox "github.com/nsf/termbox-go"
)

func main() {
	if len(os.Args) < 2 {
		panic("you must provide a path to a chip8 file")
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(fmt.Sprintf("error opening file: %v", err))
	}
	defer f.Close()

	// initialize the chip 8 system and load the game into memory
	myChip8 := NewChip8()
	myChip8.Initialize()
	myChip8.LoadGame(bufio.NewReader(f))

	termbox.Init()
	defer termbox.Close()

	exiting := false

	go func() {
		for {
			if k := termbox.PollEvent(); k.Type == termbox.EventKey && k.Key == termbox.KeyEsc {
				exiting = true
			}
		}
	}()

	for {
		if exiting {
			break
		}

		myChip8.EmulateCycle()
	}
}

func init() {
	rand.Seed(time.Now().Unix())
}
