package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/nsf/termbox-go"

	"azul3d.org/engine/keyboard"
)

var Chip8Fontset = [80]byte{
	0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
	0x20, 0x60, 0x20, 0x20, 0x70, // 1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
	0x90, 0x90, 0xF0, 0x10, 0x10, // 4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
	0xF0, 0x10, 0x20, 0x40, 0x40, // 7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
	0xF0, 0x90, 0xF0, 0x90, 0x90, // A
	0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
	0xF0, 0x80, 0x80, 0x80, 0xF0, // C
	0xE0, 0x90, 0x90, 0x90, 0xE0, // D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
	0xF0, 0x80, 0xF0, 0x80, 0x80, // F
}

type Chip8 struct {
	opcode uint16
	I      uint16
	pc     uint16
	memory [4096]byte

	V        [16]byte
	gfx      [2048]byte // 64 x 32
	drawFlag bool

	stack [16]uint16
	sp    uint16

	delayTimer uint8
	soundTimer uint8

	keys       [16]bool
	keyWatcher *keyboard.Watcher
}

func NewChip8() *Chip8 {
	return &Chip8{}
}

func (c *Chip8) Initialize() {
	c.opcode = 0
	c.I = 0
	c.sp = 0
	c.pc = 0x200 // 512
	c.memory = [4096]byte{}
	c.V = [16]byte{}
	c.gfx = [2048]byte{}
	c.stack = [16]uint16{}
	c.delayTimer = 0
	c.soundTimer = 0
	c.keys = [16]bool{}
	c.keyWatcher = keyboard.NewWatcher()
	c.drawFlag = true

	// Load fontset into the first 80 addresses of memory
	for i := 0; i < 80; i++ {
		c.memory[i] = Chip8Fontset[i]
	}
}

func (c *Chip8) LoadGame(buf *bufio.Reader) {
	i := 0x200 // 512
	for {
		b, err := buf.ReadByte()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		c.memory[i] = b
		i++
	}
}

func (c *Chip8) fetchOpcode() uint16 {
	// Merge the bytes at the current program counter and the one after it.
	return binary.BigEndian.Uint16([]byte{c.memory[c.pc], c.memory[c.pc+1]})
}

func (c *Chip8) decodeOpcode(opcode uint16) {
	// Just look at the first 4 bytes of the opcode first
	switch opcode & 0xF000 {
	// There are two cases here so switch between them
	case 0x0000:
		switch opcode & 0x000F {
		// 00E0: Clears the screen
		case 0x0000:
			c.gfx = [2048]byte{}
			c.drawFlag = true
			c.pc += 2

		// 00EE: Return from a subroutine
		case 0x000E:
			// I think to return from a subroutine we need to go back up the program stack?
			// And increment by 2 like normal
			c.sp--
			c.pc = c.stack[c.sp] + 2

		default:
			panic(fmt.Sprintf("Unknown opcode: 0x%X", opcode))
		}

	// 1NNN: Jumps to address NNN
	case 0x1000:
		c.pc = opcode & 0x0FFF
		// Don't increment the program counter as we've just jumped!

	// 2NNN: Calls subroutine at NNN
	case 0x2000:
		// temp jump to NNN, so store the current address in the stack first
		c.stack[c.sp] = c.pc
		c.sp++
		c.pc = opcode & 0x0FFF
		// Don't increment the program counter as we've just jumped!

	// 3XNN: Skips the next instruction if VX equals NN. (Usually the next instruction is a jump to skip a code block)
	case 0x3000:
		if c.V[(opcode&0x0F00)>>8] == byte(opcode&0x00FF) {
			// Skip the next instruction
			c.pc += 2
		}
		c.pc += 2

	// 4XNN: Skips the next instruction if VX doesn't equal NN. (Usually the next instruction is a jump to skip a code block)
	case 0x4000:
		if c.V[(opcode&0x0F00)>>8] != byte(opcode&0x00FF) {
			// Skip the next instruction
			c.pc += 2
		}
		c.pc += 2

	// 5XY0: Skips the next instruction if VX equals VY. (Usually the next instruction is a jump to skip a code block)
	case 0x5000:
		if c.V[(opcode&0x0F00)>>8] == c.V[(opcode&0x00F0)>>4] {
			// Skip the next instruction
			c.pc += 2
		}
		c.pc += 2

	// 6XNN: Sets VX to NN.
	case 0x6000:
		c.V[(opcode&0x0F00)>>8] = byte(opcode & 0x00FF)
		c.pc += 2

	// 7XNN: Adds NN to VX. (Carry flag is not changed)
	case 0x7000:
		c.V[(opcode&0x0F00)>>8] += byte(opcode & 0x00FF)
		c.pc += 2

	case 0x8000:
		switch opcode & 0x000F {
		// 8XY0: Sets VX to the value of VY.
		case 0x0000:
			c.V[(opcode&0x0F00)>>8] = c.V[(opcode&0x00F0)>>4]
			c.pc += 2

		// 8XY1: Sets VX to VX or VY. (Bitwise OR operation)
		case 0x0001:
			c.V[(opcode&0x0F00)>>8] = c.V[(opcode&0x0F00)>>8] | c.V[(opcode&0x00F0)>>4]
			c.pc += 2

		// 8XY2: Sets VX to VX and VY. (Bitwise AND operation)
		case 0x0002:
			c.V[(opcode&0x0F00)>>8] = c.V[(opcode&0x0F00)>>8] & c.V[(opcode&0x00F0)>>4]
			c.pc += 2

		// 8XY3: Sets VX to VX xor VY.
		case 0x0003:
			c.V[(opcode&0x0F00)>>8] = c.V[(opcode&0x0F00)>>8] ^ c.V[(opcode&0x00F0)>>4]
			c.pc += 2

		// 8XY4: Adds VY to VX. VF is set to 1 when there's a carry, and to 0 when there isn't.
		case 0x0004:
			// Explanation on Opcode Example 2 here http://www.multigesture.net/articles/how-to-write-an-emulator-chip-8-interpreter/
			if c.V[(opcode&0x00F0)>>4] > (0xFF - c.V[(opcode&0x0F00)>>8]) {
				c.V[0xF] = 1 //carry
			} else {
				c.V[0xF] = 0
			}
			c.V[(opcode&0x0F00)>>8] += c.V[(opcode&0x00F0)>>4]
			c.pc += 2

		// 8XY5: VY is subtracted from VX. VF is set to 0 when there's a borrow, and 1 when there isn't.
		case 0x0005:
			vx := c.V[(opcode&0x0F00)>>8]
			vy := c.V[(opcode&0x00F0)>>4]
			if vx-vy < 0 {
				c.V[0xF] = 1 // borrow
			} else {
				c.V[0xF] = 0
			}
			c.V[(opcode&0x0F00)>>8] = vx - vy
			c.pc += 2

		// 8XY6: Stores the least significant bit of VX in VF and then shifts VX to the right by 1.
		case 0x0006:
			c.V[0xF] = c.V[(opcode&0x0F00)>>8] & 0x0000000F
			c.V[(opcode&0x0F00)>>8] = c.V[(opcode&0x0F00)>>8] >> 1
			c.pc += 2

		// 8XY7: Sets VX to VY minus VX. VF is set to 0 when there's a borrow, and 1 when there isn't.
		case 0x0007:
			vx := c.V[(opcode&0x0F00)>>8]
			vy := c.V[(opcode&0x00F0)>>4]
			if vy-vx < 0 {
				c.V[0xF] = 0 // borrow
			} else {
				c.V[0xF] = 1
			}
			c.V[(opcode&0x00F0)>>4] = vy - vx
			c.pc += 2

		// 8XYE: Stores the most significant bit of VX in VF and then shifts VX to the left by 1.
		case 0x000E:
			c.V[0xF] = c.V[(opcode&0x0F00)>>8] >> 7
			c.V[(opcode&0x0F00)>>8] <<= 1
			c.pc += 2

		default:
			panic(fmt.Sprintf("Unknown opcode: 0x%X", opcode))
		}

	// 9XY0: Skips the next instruction if VX doesn't equal VY. (Usually the next instruction is a jump to skip a code block)
	case 0x9000:
		if c.V[(opcode&0x0F00)>>8] != c.V[(opcode&0x00F0)>>4] {
			c.pc += 2
		}
		c.pc += 2

	// ANNN: Sets i to the address NNN
	case 0xA000:
		c.I = opcode & 0x0FFF
		c.pc += 2

	// BNNN: Jumps to the address NNN plus V0.
	case 0xB000:
		c.pc = (opcode & 0x0FFF) + uint16(c.V[0])
		// Don't increment the program counter as we've just jumped

	// CXNN: Sets VX to the result of a bitwise and operation on a random number (Typically: 0 to 255) and NN.
	case 0xC000:
		r := byte(rand.Intn(256))
		c.V[(opcode&0x0F00)>>8] = r & byte(opcode&0x00FF)
		c.pc += 2

	// DXYN: Draws a sprite at coordinate (VX, VY) that has a width of 8 pixels and a height of N pixels.
	// Each row of 8 pixels is read as bit-coded starting from memory location I; I value doesn’t change
	// after the execution of this instruction. As described above, VF is set to 1 if any screen pixels
	// are flipped from set to unset when the sprite is drawn, and to 0 if that doesn’t happen
	case 0xD000:
		x := c.V[(opcode&0x0F00)>>8]
		y := c.V[(opcode&0x00F0)>>4]
		height := opcode & 0x000F

		// First reset VF
		c.V[0xF] = 0
		idx := uint16(0)
		defer func() {
			if r := recover(); r != nil {
				panic(fmt.Sprintf("attempted to access index %v. x %v, y %v, height %v", idx, x, y, height))
			}
		}()

		for yline := uint16(0); yline < height; yline++ {
			pixel := c.memory[c.I+yline]
			for xline := uint16(0); xline < 8; xline++ {
				if pixel&(0x80>>xline) != 0 {
					idx = uint16(x) + xline + ((uint16(y) + yline) * 64)
					if c.gfx[idx] == 1 {
						c.V[0xF] = 1
					}

					c.gfx[idx] ^= 1
				}
			}
		}
		c.drawFlag = true
		c.pc += 2

	case 0xE000:
		switch opcode & 0x00FF {
		// EX9E: Skips the next instruction if the key stored in VX is pressed. (Usually the next instruction is a jump to skip a code block)
		case 0x009E:
			if c.keys[(opcode&0x0F00)>>8] {
				c.pc += 2
			}
			c.pc += 2

		// EXA1: Skips the next instruction if the key stored in VX isn't pressed. (Usually the next instruction is a jump to skip a code block)
		case 0x00A1:
			if !c.keys[(opcode&0x0F00)>>8] {
				c.pc += 2
			}
			c.pc += 2

		default:
			panic(fmt.Sprintf("Unknown opcode: 0x%X", opcode))
		}

	case 0xF000:
		switch opcode & 0x00FF {
		// FX07: Sets VX to the value of the delay timer.
		case 0x0007:
			c.V[(opcode&0x0F00)>>8] = c.delayTimer
			c.pc += 2

		// FX0A: A key press is awaited, and then stored in VX. (Blocking Operation. All instruction halted until next key event)
		case 0x000A:
			newKey := c.awaitKeyPress()
			c.V[(opcode&0x0F00)>>8] = newKey
			c.pc += 2

		// FX15: Sets the delay timer to VX.
		case 0x0015:
			c.delayTimer = c.V[(opcode&0x0F00)>>8]
			c.pc += 2

		// FX18: Sets the sound timer to VX.
		case 0x0018:
			c.soundTimer = c.V[(opcode&0x0F00)>>8]
			c.pc += 2

		// FX1E: Adds VX to I.
		case 0x001E:
			c.I += uint16(c.V[(opcode&0x0F00)>>8])
			c.pc += 2

		// FX29: Sets I to the location of the sprite for the character in VX. Characters 0-F (in hexadecimal) are represented by a 4x5 font.
		case 0x0029:
			c.I = uint16(c.V[(opcode&0x0F00)>>8] * 0x5)
			c.pc += 2

		// FX33: Stores the binary-coded decimal representation of VX, with the most significant of three digits at the address in I,
		// the middle digit at I plus 1, and the least significant digit at I plus 2. (In other words, take the decimal
		// representation of VX, place the hundreds digit in memory at location in I, the tens digit at location I+1, and the ones digit at location I+2.)
		case 0x0033:
			// Taken from http://www.multigesture.net/wp-content/uploads/mirror/goldroad/chip8.shtml
			c.memory[c.I] = c.V[(opcode&0x0F00)>>8] / 100
			c.memory[c.I+1] = (c.V[(opcode&0x0F00)>>8] / 10) % 10
			c.memory[c.I+2] = (c.V[(opcode&0x0F00)>>8] % 100) % 10
			c.pc += 2

		// FX55: Stores V0 to VX (including VX) in memory starting at address I.
		// The offset from I is increased by 1 for each value written, but I itself is left unmodified.
		case 0x0055:
			x := (opcode & 0x0F00) >> 8
			for i := uint16(0); i < x; i++ {
				c.memory[c.I+i] = c.V[i]
			}
			// On the original interpreter, when the operation is done, I = I + X + 1.
			c.I += x + 1
			c.pc += 2

		// FX65: Fills V0 to VX (including VX) with values from memory starting at address I.
		// The offset from I is increased by 1 for each value written, but I itself is left unmodified.
		case 0x0065:
			x := (opcode & 0x0F00) >> 8
			for i := uint16(0); i < x; i++ {
				c.V[i] = c.memory[c.I+i]
			}
			// On the original interpreter, when the operation is done, I = I + X + 1.
			c.I += x + 1
			c.pc += 2

		default:
			panic(fmt.Sprintf("Unknown opcode: 0x%X", opcode))
		}

	default:
		panic(fmt.Sprintf("Unknown opcode: 0x%X", opcode))
	}
}

func (c *Chip8) getKeyState() [16]bool {
	keys := [16]bool{
		c.keyWatcher.Down(keyboard.One),
		c.keyWatcher.Down(keyboard.Two),
		c.keyWatcher.Down(keyboard.Three),
		c.keyWatcher.Down(keyboard.Four),
		c.keyWatcher.Down(keyboard.Q),
		c.keyWatcher.Down(keyboard.W),
		c.keyWatcher.Down(keyboard.E),
		c.keyWatcher.Down(keyboard.R),
		c.keyWatcher.Down(keyboard.A),
		c.keyWatcher.Down(keyboard.S),
		c.keyWatcher.Down(keyboard.D),
		c.keyWatcher.Down(keyboard.F),
		c.keyWatcher.Down(keyboard.Z),
		c.keyWatcher.Down(keyboard.X),
		c.keyWatcher.Down(keyboard.C),
		c.keyWatcher.Down(keyboard.V),
	}

	if c.keyWatcher.Down(keyboard.Escape) {
		os.Exit(1)
	}

	return keys
}

func (c *Chip8) awaitKeyPress() (keyIdx uint8) {
	for {
		// Get the current key state every 1/60th of a second
		newKeys := c.getKeyState()
		for i := uint8(0); i < 16; i++ {
			if newKeys[i] && newKeys[i] != c.keys[i] {
				// Newly pressed key, return it. Set the key state first
				c.keys = newKeys
				return i
			}
		}
		time.Sleep(time.Second / 60)
	}
}

func (c *Chip8) drawGraphics() {
	// tm.Clear()

	// for y := 0; y < 32; y++ {
	// 	for x := 0; x < 64; x++ {
	// 		if c.gfx[(y*64)+x] == 1 {
	// 			tm.Printf(tm.Background(" ", tm.WHITE))
	// 		} else {
	// 			tm.Printf(tm.Background(" ", tm.BLACK))
	// 		}
	// 	}
	// 	tm.Printf("\n")
	// }
	// tm.Printf("\n")
	// tm.Flush()

	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	for y := 0; y < 32; y++ {
		for x := 0; x < 64; x++ {
			if c.gfx[(y*64)+x] == 1 {
				termbox.SetCell(x, y, ' ', termbox.ColorDefault, termbox.ColorWhite)
			} else {
				termbox.SetCell(x, y, ' ', termbox.ColorDefault, termbox.ColorBlack)
			}
		}
	}
	termbox.Flush()
}

func (c *Chip8) EmulateCycle() {
	// First fetch the current opcode.
	opcode := c.fetchOpcode()

	// Next decode it
	c.decodeOpcode(opcode)

	// Draw
	if c.drawFlag {
		c.drawFlag = false
		c.drawGraphics()
	}

	c.keys = c.getKeyState()

	// And update timers
	if c.delayTimer > 0 {
		c.delayTimer--
	}

	if c.soundTimer > 0 {
		if c.soundTimer == 1 {
			fmt.Printf("BEEP!!\n")
		}
		c.soundTimer--
	}

	time.Sleep(time.Second / 540)
}
