package main

import "bytes"

func main() {
	// setupGraphics()
	// setupInput()

	// initialize the chip 8 system and load the game into memory
	myChip8 := NewChip8()
	myChip8.LoadGame(bytes.NewBuffer([]byte{}))
}
