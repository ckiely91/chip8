package main

import (
	"bytes"
	"math/rand"
	"time"
)

func main() {
	// setupGraphics()
	// setupInput()

	// initialize the chip 8 system and load the game into memory
	myChip8 := NewChip8()
	myChip8.LoadGame(bytes.NewBuffer([]byte{}))
}

func init() {
	rand.Seed(time.Now().Unix())
}
