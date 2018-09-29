package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
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
	i      uint16
	pc     uint16
	memory [4096]byte

	registers [16]byte
	gfx       [32][64]bool

	stack [16]uint16
	sp    uint16

	delayTimer uint8
	soundTimer uint8

	key [16]uint8
}

func NewChip8() *Chip8 {
	return &Chip8{}
}

func (c *Chip8) Initialize() {
	c.opcode = 0
	c.i = 0
	c.sp = 0
	c.pc = 0x200 // 512
	c.memory = [4096]byte{}
	c.registers = [16]byte{}
	c.gfx = [32][64]bool{}
	c.stack = [16]uint16{}
	c.delayTimer = 0
	c.soundTimer = 0

	// Load fontset into the first 80 addresses of memory
	for i := 0; i < 80; i++ {
		c.memory[i] = Chip8Fontset[i]
	}
}

func (c *Chip8) LoadGame(buf *bytes.Buffer) {
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
			// TODO: clear the screen

			// 00EE: Return from a subroutine
		case 0x000E:
			// TODO: return from a subroutine
		default:
			panic(fmt.Sprintf("Unknown opcode: 0x%X", opcode))
		}

	// 1NNN: Jumps to address NNN
	case 0x1000:
		// TODO: Jump to address NNN

	// 2NNN: Calls subroutine at NNN
	case 0x2000:
		// TODO: call subroutine at NNN

	// 3XNN: Skips the next instruction if VX equals NN. (Usually the next instruction is a jump to skip a code block)
	case 0x3000:
		// TODO

	// 4XNN: Skips the next instruction if VX doesn't equal NN. (Usually the next instruction is a jump to skip a code block)
	case 0x4000:
		// TODO

	// 5XY0: Skips the next instruction if VX equals VY. (Usually the next instruction is a jump to skip a code block)
	case 0x5000:
		// TODO

	// 6XNN: Sets VX to NN.
	case 0x6000:
		// TODO

	// 7XNN: Adds NN to VX. (Carry flag is not changed)
	case 0x7000:
		// TODO

	case 0x8000:
		switch opcode & 0x000F {
		// 8XY0: Sets VX to the value of VY.
		case 0x0000:
			// TODO

		// 8XY1: Sets VX to VX or VY. (Bitwise OR operation)
		case 0x0001:
			// TODO

		// 8XY2: Sets VX to VX and VY. (Bitwise AND operation)
		case 0x0002:
			// TODO

		// 8XY3: Sets VX to VX xor VY.
		case 0x0003:
			// TODO

		// 8XY4: Adds VY to VX. VF is set to 1 when there's a carry, and to 0 when there isn't.
		case 0x0004:
			// TODO

		// 8XY5: VY is subtracted from VX. VF is set to 0 when there's a borrow, and 1 when there isn't.
		case 0x0005:
			// TODO

		// 8XY6: Stores the least significant bit of VX in VF and then shifts VX to the right by 1.
		case 0x0006:
			// TODO

		// 8XY7: Sets VX to VY minus VX. VF is set to 0 when there's a borrow, and 1 when there isn't.
		case 0x0007:
			// TODO

		// 8XYE: Stores the most significant bit of VX in VF and then shifts VX to the left by 1.
		case 0x000E:
			// TODO
		}

	// 9XY0: Skips the next instruction if VX doesn't equal VY. (Usually the next instruction is a jump to skip a code block)
	case 0x9000:
		// TODO

	// ANNN: Sets i to the address NNN
	case 0xA000:
		c.i = opcode & 0x0FFF
		c.pc += 2

	// BNNN: Jumps to the address NNN plus V0.
	case 0xB000:
		// TODO

	// CXNN: Sets VX to the result of a bitwise and operation on a random number (Typically: 0 to 255) and NN.
	case 0xC000:
		// TODO

	// DXYN: Draws a sprite at coordinate (VX, VY) that has a width of 8 pixels and a height of N pixels.
	// Each row of 8 pixels is read as bit-coded starting from memory location I; I value doesn’t change
	// after the execution of this instruction. As described above, VF is set to 1 if any screen pixels
	// are flipped from set to unset when the sprite is drawn, and to 0 if that doesn’t happen
	case 0xD000:
		// TODO

	case 0xE000:
		switch opcode & 0x00FF {
		// EX9E: Skips the next instruction if the key stored in VX is pressed. (Usually the next instruction is a jump to skip a code block)
		case 0x009E:
			// TODO

		// EXA1: Skips the next instruction if the key stored in VX isn't pressed. (Usually the next instruction is a jump to skip a code block)
		case 0x00A1:
			// TODO

		default:
			panic(fmt.Sprintf("Unknown opcode: 0x%X", opcode))
		}

	case 0xF000:
		switch opcode & 0x00FF {
		// FX07: Sets VX to the value of the delay timer.
		case 0x0007:
			// TODO

		// FX0A: A key press is awaited, and then stored in VX. (Blocking Operation. All instruction halted until next key event)
		case 0x000A:
			// TODO

		// FX15: Sets the delay timer to VX.
		case 0x0015:
			// TODO

		// FX18: Sets the sound timer to VX.
		case 0x0018:
			// TODO

		// FX1E: Adds VX to I.
		case 0x001E:
			// TODO

		// FX29: Sets I to the location of the sprite for the character in VX. Characters 0-F (in hexadecimal) are represented by a 4x5 font.
		case 0x0029:
			// TODO

		// FX33: Stores the binary-coded decimal representation of VX, with the most significant of three digits at the address in I,
		// the middle digit at I plus 1, and the least significant digit at I plus 2. (In other words, take the decimal
		// representation of VX, place the hundreds digit in memory at location in I, the tens digit at location I+1, and the ones digit at location I+2.)
		case 0x0033:
			// TODO

		// FX55: Stores V0 to VX (including VX) in memory starting at address I.
		// The offset from I is increased by 1 for each value written, but I itself is left unmodified.
		case 0x0055:
			// TODO

		// FX65: Fills V0 to VX (including VX) with values from memory starting at address I.
		// The offset from I is increased by 1 for each value written, but I itself is left unmodified.
		case 0x0065:
			// TODO

		default:
			panic(fmt.Sprintf("Unknown opcode: 0x%X", opcode))
		}

	default:
		panic(fmt.Sprintf("Unknown opcode: 0x%X", opcode))
	}
}

func (c *Chip8) EmulateCycle() {
	// First fetch the current opcode.
	opcode := c.fetchOpcode()

	// Next decode it
	c.decodeOpcode(opcode)

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
}
