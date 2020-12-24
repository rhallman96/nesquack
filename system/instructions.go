package system

type operation func(c *cpu, bus memoryDevice, a uint16) error

type instruction struct {
	operation   operation
	addressMode addressMode
	cycles      uint64
}

// execute modifies the state of the cpu and memory on behalf of an instruction
func (i *instruction) execute(c *cpu, bus memoryDevice) error {
	c.pc++
	a, err := i.addressMode(c, bus)
	if err != nil {
		return err
	}
	c.clock += i.cycles
	return i.operation(c, bus, a)
}

// the 2A03 instruction set, excluding undocumented opcodes
var instructionSet = [256]*instruction{
	// ADC
	0x69: {adc, immediate, 2},
	0x65: {adc, zeroPage, 3},
	0x75: {adc, zeroPageX, 4},
	0x6d: {adc, absolute, 4},
	0x7d: {adc, absoluteX, 4},
	0x79: {adc, absoluteY, 4},
	0x61: {adc, indirectX, 6},
	0x71: {adc, indirectY, 5},

	// AND
	0x29: {and, immediate, 2},
	0x25: {and, zeroPage, 3},
	0x35: {and, zeroPageX, 4},
	0x2d: {and, absolute, 4},
	0x3d: {and, absoluteX, 4},
	0x39: {and, absoluteY, 4},
	0x21: {and, indirectX, 6},
	0x31: {and, indirectY, 5},

	// ASL
	0x0a: {aslAcc, accumulator, 2},
	0x06: {asl, zeroPage, 5},
	0x16: {asl, zeroPageX, 6},
	0x0e: {asl, absolute, 6},
	0x1e: {asl, absoluteX, 7},

	// Branch
	0x90: {bcc, relative, 2},
	0xb0: {bcs, relative, 2},
	0xf0: {beq, relative, 2},
	0x30: {bmi, relative, 2},
	0xd0: {bne, relative, 2},
	0x10: {bpl, relative, 2},
	0x50: {bvc, relative, 2},
	0x70: {bvs, relative, 2},

	// BIT
	0x24: {bit, zeroPage, 3},
	0x2c: {bit, absolute, 4},

	// Break
	0x00: {brk, implied, 7},

	// Clear
	0x18: {clc, implied, 2},
	0xd8: {cld, implied, 2},
	0x58: {cli, implied, 2},
	0xb8: {clv, implied, 2},

	// CMP
	0xc9: {cmp, immediate, 2},
	0xc5: {cmp, zeroPage, 3},
	0xd5: {cmp, zeroPageX, 4},
	0xcd: {cmp, absolute, 4},
	0xdd: {cmp, absoluteX, 4},
	0xd9: {cmp, absoluteY, 4},
	0xc1: {cmp, indirectX, 6},
	0xd1: {cmp, indirectY, 5},

	// CPX
	0xe0: {cpx, immediate, 2},
	0xe4: {cpx, zeroPage, 3},
	0xec: {cpx, absolute, 4},

	// CPY
	0xc0: {cpy, immediate, 2},
	0xc4: {cpy, zeroPage, 3},
	0xcc: {cpy, absolute, 4},

	// DEC
	0xc6: {dec, zeroPage, 5},
	0xd6: {dec, zeroPageX, 6},
	0xce: {dec, absolute, 6},
	0xde: {dec, absoluteX, 7},

	// Dec Registers
	0xca: {dex, implied, 2},
	0x88: {dey, implied, 2},

	// EOR
	0x49: {eor, immediate, 2},
	0x45: {eor, zeroPage, 3},
	0x55: {eor, zeroPageX, 4},
	0x4d: {eor, absolute, 4},
	0x5d: {eor, absoluteX, 4},
	0x59: {eor, absoluteY, 4},
	0x41: {eor, indirectX, 6},
	0x51: {eor, indirectY, 5},

	// INC
	0xe6: {inc, zeroPage, 5},
	0xf6: {inc, zeroPageX, 6},
	0xee: {inc, absolute, 6},
	0xfe: {inc, absoluteX, 7},

	// Inc Registers
	0xe8: {inx, implied, 2},
	0xc8: {iny, implied, 2},

	// JMP
	0x4c: {jmp, absolute, 3},
	0x6c: {jmp, indirect, 5},

	// JSR
	0x20: {jsr, absolute, 6},

	// LDA
	0xa9: {lda, immediate, 2},
	0xa5: {lda, zeroPage, 3},
	0xb5: {lda, zeroPageX, 4},
	0xad: {lda, absolute, 4},
	0xbd: {lda, absoluteX, 4},
	0xb9: {lda, absoluteY, 4},
	0xa1: {lda, indirectX, 6},
	0xb1: {lda, indirectY, 5},

	// LDX
	0xa2: {ldx, immediate, 2},
	0xa6: {ldx, zeroPage, 3},
	0xb6: {ldx, zeroPageY, 4},
	0xae: {ldx, absolute, 4},
	0xbe: {ldx, absoluteY, 4},

	// LDY
	0xa0: {ldy, immediate, 2},
	0xa4: {ldy, zeroPage, 3},
	0xb4: {ldy, zeroPageX, 4},
	0xac: {ldy, absolute, 4},
	0xbc: {ldy, absoluteX, 4},

	// LSR
	0x4a: {lsrAcc, accumulator, 2},
	0x46: {lsr, zeroPage, 5},
	0x56: {lsr, zeroPageX, 6},
	0x4e: {lsr, absolute, 6},
	0x5e: {lsr, absoluteX, 7},

	// NOP
	0xea: {nop, implied, 2},

	// ORA
	0x09: {ora, immediate, 2},
	0x05: {ora, zeroPage, 3},
	0x15: {ora, zeroPageX, 4},
	0x0d: {ora, absolute, 4},
	0x1d: {ora, absoluteX, 4},
	0x19: {ora, absoluteY, 4},
	0x01: {ora, indirectX, 6},
	0x11: {ora, indirectY, 5},

	// Push
	0x48: {pha, implied, 3},
	0x08: {php, implied, 3},

	// Pull
	0x68: {pla, implied, 4},
	0x28: {plp, implied, 4},

	// ROL
	0x2a: {rolAcc, accumulator, 2},
	0x26: {rol, zeroPage, 5},
	0x36: {rol, zeroPageX, 6},
	0x2e: {rol, absolute, 6},
	0x3e: {rol, absoluteX, 7},

	// ROR
	0x6a: {rorAcc, accumulator, 2},
	0x66: {ror, zeroPage, 5},
	0x76: {ror, zeroPageX, 6},
	0x6e: {ror, absolute, 6},
	0x7e: {ror, absoluteX, 7},

	// RTI
	0x40: {rti, implied, 6},

	// RTS
	0x60: {rts, implied, 6},

	// SBC
	0xe9: {sbc, immediate, 2},
	0xe5: {sbc, zeroPage, 3},
	0xf5: {sbc, zeroPageX, 4},
	0xed: {sbc, absolute, 4},
	0xfd: {sbc, absoluteX, 4},
	0xf9: {sbc, absoluteY, 4},
	0xe1: {sbc, indirectX, 6},
	0xf1: {sbc, indirectY, 5},

	// Set flags
	0x38: {sec, implied, 2},
	0xf8: {sed, implied, 2},
	0x78: {sei, implied, 2},

	// STA
	0x85: {sta, zeroPage, 3},
	0x95: {sta, zeroPageX, 4},
	0x8d: {sta, absolute, 4},
	0x9d: {sta, absoluteX, 5},
	0x99: {sta, absoluteY, 5},
	0x81: {sta, indirectX, 6},
	0x91: {sta, indirectY, 6},

	// STX
	0x86: {stx, zeroPage, 3},
	0x96: {stx, zeroPageY, 4},
	0x8e: {stx, absolute, 4},

	// STY
	0x84: {sty, zeroPage, 3},
	0x94: {sty, zeroPageX, 4},
	0x8c: {sty, absolute, 4},

	// Transfers
	0xaa: {tax, implied, 2},
	0xa8: {tay, implied, 2},
	0xba: {tsx, implied, 2},
	0x8a: {txa, implied, 2},
	0x9a: {txs, implied, 2},
	0x98: {tya, implied, 2},
}

func adc(c *cpu, bus memoryDevice, a uint16) error {
	v, err := bus.read(a)
	if err != nil {
		return err
	}
	tmp := uint16(v) + uint16(c.a)
	if c.isFlagSet(flagCarry) {
		tmp++
	}
	c.setZeroFlag(uint8(tmp))
	c.setSignFlag(uint8(tmp))
	c.setFlagValue(flagCarry, (tmp > 0xff))

	o := (((c.a ^ v) & 0x80) == 0) && (((c.a ^ uint8(tmp)) & 0x80) != 0)
	c.setFlagValue(flagOverflow, o)

	c.a = uint8(tmp)
	return nil
}

func and(c *cpu, bus memoryDevice, a uint16) error {
	v, err := bus.read(a)
	if err != nil {
		return err
	}
	c.setSignFlag(v)
	c.setZeroFlag(v)
	c.a = v
	return nil
}

func asl(c *cpu, bus memoryDevice, a uint16) error {
	v, err := bus.read(a)
	if err != nil {
		return err
	}
	c.setFlagValue(flagCarry, (v&0x80) != 0)
	v <<= 1
	c.setSignFlag(v)
	c.setZeroFlag(v)
	return bus.write(a, v)
}

func aslAcc(c *cpu, bus memoryDevice, a uint16) error {
	c.setFlagValue(flagCarry, (c.a&0x80) != 0)
	c.a <<= 1
	c.setSignFlag(c.a)
	c.setZeroFlag(c.a)
	return nil
}

func bcc(c *cpu, bus memoryDevice, a uint16) error {
	if !c.isFlagSet(flagCarry) {
		branch(c, a)
	}
	return nil
}

func bcs(c *cpu, bus memoryDevice, a uint16) error {
	if c.isFlagSet(flagCarry) {
		branch(c, a)
	}
	return nil
}

func beq(c *cpu, bus memoryDevice, a uint16) error {
	if c.isFlagSet(flagZero) {
		branch(c, a)
	}
	return nil
}

func bit(c *cpu, bus memoryDevice, a uint16) error {
	v, err := bus.read(a)
	if err != nil {
		return err
	}
	c.setSignFlag(v)
	c.setZeroFlag(v & c.a)
	c.setFlagValue(flagOverflow, (0x40&v) != 0)
	return err
}

func bmi(c *cpu, bus memoryDevice, a uint16) error {
	if c.isFlagSet(flagSign) {
		branch(c, a)
	}
	return nil
}

func bne(c *cpu, bus memoryDevice, a uint16) error {
	if !c.isFlagSet(flagZero) {
		branch(c, a)
	}
	return nil
}

func bpl(c *cpu, bus memoryDevice, a uint16) error {
	if !c.isFlagSet(flagSign) {
		branch(c, a)
	}
	return nil
}

func brk(c *cpu, bus memoryDevice, a uint16) error {
	c.setFlag(flagBreak)
	return c.branch(bus, 0xfffe)
}

func bvc(c *cpu, bus memoryDevice, a uint16) error {
	if !c.isFlagSet(flagOverflow) {
		branch(c, a)
	}
	return nil
}

func bvs(c *cpu, bus memoryDevice, a uint16) error {
	if c.isFlagSet(flagOverflow) {
		branch(c, a)
	}
	return nil
}

func clc(c *cpu, bus memoryDevice, a uint16) error {
	c.setFlagValue(flagCarry, false)
	return nil
}

func cld(c *cpu, bus memoryDevice, a uint16) error {
	c.setFlagValue(flagDecimal, false)
	return nil
}

func cli(c *cpu, bus memoryDevice, a uint16) error {
	c.setFlagValue(flagDisableIRQ, false)
	return nil
}

func clv(c *cpu, bus memoryDevice, a uint16) error {
	c.setFlagValue(flagOverflow, false)
	return nil
}

func cmp(c *cpu, bus memoryDevice, a uint16) error {
	v, err := bus.read(a)
	if err != nil {
		return err
	}
	c.setFlagValue(flagCarry, (v <= c.a))
	v = c.a - v
	c.setSignFlag(v)
	c.setZeroFlag(v)
	return nil
}

func cpx(c *cpu, bus memoryDevice, a uint16) error {
	v, err := bus.read(a)
	if err != nil {
		return err
	}
	c.setFlagValue(flagCarry, (v <= c.x))
	v = c.x - v
	c.setSignFlag(v)
	c.setZeroFlag(v)
	return nil
}

func cpy(c *cpu, bus memoryDevice, a uint16) error {
	v, err := bus.read(a)
	if err != nil {
		return err
	}
	c.setFlagValue(flagCarry, (v <= c.y))
	v = c.y - v
	c.setSignFlag(v)
	c.setZeroFlag(v)
	return nil
}

func dec(c *cpu, bus memoryDevice, a uint16) error {
	v, err := bus.read(a)
	if err != nil {
		return err
	}
	v--
	c.setSignFlag(v)
	c.setZeroFlag(v)
	return bus.write(a, v)
}

func dex(c *cpu, bus memoryDevice, a uint16) error {
	c.x--
	c.setSignFlag(c.x)
	c.setZeroFlag(c.x)
	return nil
}

func dey(c *cpu, bus memoryDevice, a uint16) error {
	c.y--
	c.setSignFlag(c.y)
	c.setZeroFlag(c.y)
	return nil
}

func eor(c *cpu, bus memoryDevice, a uint16) error {
	v, err := bus.read(a)
	if err != nil {
		return err
	}
	c.a ^= v
	c.setSignFlag(c.a)
	c.setZeroFlag(c.a)
	return nil
}

func inc(c *cpu, bus memoryDevice, a uint16) error {
	v, err := bus.read(a)
	if err != nil {
		return err
	}
	v++
	c.setSignFlag(v)
	c.setZeroFlag(v)
	return bus.write(a, v)
}

func inx(c *cpu, bus memoryDevice, a uint16) error {
	c.x++
	c.setSignFlag(c.x)
	c.setZeroFlag(c.x)
	return nil
}

func iny(c *cpu, bus memoryDevice, a uint16) error {
	c.y++
	c.setSignFlag(c.y)
	c.setZeroFlag(c.y)
	return nil
}

func jmp(c *cpu, bus memoryDevice, a uint16) error {
	c.pc = a
	return nil
}

func jsr(c *cpu, bus memoryDevice, a uint16) error {
	c.pc--
	err := c.pushWord(bus, c.pc)
	if err != nil {
		return err
	}
	c.pc = a
	return nil
}

func lda(c *cpu, bus memoryDevice, a uint16) error {
	v, err := bus.read(a)
	if err != nil {
		return err
	}
	c.a = v
	c.setSignFlag(c.a)
	c.setZeroFlag(c.a)
	return nil
}

func ldx(c *cpu, bus memoryDevice, a uint16) error {
	v, err := bus.read(a)
	if err != nil {
		return err
	}
	c.x = v
	c.setSignFlag(c.x)
	c.setZeroFlag(c.x)
	return nil
}

func ldy(c *cpu, bus memoryDevice, a uint16) error {
	v, err := bus.read(a)
	if err != nil {
		return err
	}
	c.y = v
	c.setSignFlag(c.y)
	c.setZeroFlag(c.y)
	return nil
}

func lsr(c *cpu, bus memoryDevice, a uint16) error {
	v, err := bus.read(a)
	if err != nil {
		return err
	}
	c.setFlagValue(flagCarry, (v&0x01) != 0)
	v >>= 1
	c.setSignFlag(v)
	c.setZeroFlag(v)
	return bus.write(a, v)
}

func lsrAcc(c *cpu, bus memoryDevice, a uint16) error {
	c.setFlagValue(flagCarry, (c.a&0x01) != 0)
	c.a >>= 1
	c.setSignFlag(c.a)
	c.setZeroFlag(c.a)
	return nil
}

func nop(c *cpu, bus memoryDevice, a uint16) error {
	return nil
}

func ora(c *cpu, bus memoryDevice, a uint16) error {
	v, err := bus.read(a)
	if err != nil {
		return err
	}
	c.a |= v
	c.setSignFlag(c.a)
	c.setZeroFlag(c.a)
	return nil
}

func pha(c *cpu, bus memoryDevice, a uint16) error {
	return c.push(bus, c.a)
}

func php(c *cpu, bus memoryDevice, a uint16) error {
	c.setFlag(flagBreakHi)
	c.setFlag(flagBreak)
	return c.push(bus, c.p)
}

func pla(c *cpu, bus memoryDevice, a uint16) error {
	v, err := c.pull(bus)
	if err != nil {
		return err
	}
	c.a = v
	c.setSignFlag(c.a)
	c.setZeroFlag(c.a)
	return nil
}

func plp(c *cpu, bus memoryDevice, a uint16) error {
	v, err := c.pull(bus)
	if err != nil {
		return err
	}
	c.p = v
	c.setSignFlag(c.p)
	c.setZeroFlag(c.p)
	return nil
}

func rol(c *cpu, bus memoryDevice, a uint16) error {
	v, err := bus.read(a)
	if err != nil {
		return err
	}
	var carry bool = ((v & 0x80) != 0)
	v <<= 1
	if c.isFlagSet(flagCarry) {
		v |= 0x01
	}
	c.setFlagValue(flagCarry, carry)
	c.setSignFlag(v)
	c.setZeroFlag(v)
	return bus.write(a, v)
}

func rolAcc(c *cpu, bus memoryDevice, a uint16) error {
	var carry bool = ((c.a & 0x80) != 0)
	c.a <<= 1
	if c.isFlagSet(flagCarry) {
		c.a |= 0x01
	}
	c.setFlagValue(flagCarry, carry)
	c.setSignFlag(c.a)
	c.setZeroFlag(c.a)
	return nil
}

func ror(c *cpu, bus memoryDevice, a uint16) error {
	v, err := bus.read(a)
	if err != nil {
		return err
	}
	var carry bool = ((v & 0x01) != 0)
	v >>= 1
	if c.isFlagSet(flagCarry) {
		v |= 0x80
	}
	c.setFlagValue(flagCarry, carry)
	c.setSignFlag(v)
	c.setZeroFlag(v)
	return bus.write(a, v)
}

func rorAcc(c *cpu, bus memoryDevice, a uint16) error {
	var carry bool = ((c.a & 0x01) != 0)
	c.a >>= 1
	if c.isFlagSet(flagCarry) {
		c.a |= 0x80
	}
	c.setFlagValue(flagCarry, carry)
	c.setSignFlag(c.a)
	c.setZeroFlag(c.a)
	return nil
}

func rti(c *cpu, bus memoryDevice, a uint16) error {
	v, err := c.pull(bus)
	if err != nil {
		return err
	}
	c.p = v
	addr, err := c.pullWord(bus)
	if err != nil {
		return err
	}
	c.pc = addr
	return nil
}

func rts(c *cpu, bus memoryDevice, a uint16) error {
	v, err := c.pullWord(bus)
	if err != nil {
		return err
	}
	c.pc = v + 1
	return nil
}

func sbc(c *cpu, bus memoryDevice, a uint16) error {
	v, err := bus.read(a)
	if err != nil {
		return err
	}
	tmp := uint16(c.a) - uint16(v)
	if c.isFlagSet(flagCarry) {
		tmp--
	}
	c.setSignFlag(uint8(tmp))
	c.setZeroFlag(uint8(tmp))
	c.setFlagValue(flagCarry, tmp < 0x100)

	o := ((c.a^uint8(tmp))&0x80 != 0) && ((c.a^uint8(tmp))&0x80 != 0)
	c.setFlagValue(flagOverflow, o)

	c.a = uint8(tmp)
	return nil
}

func sec(c *cpu, bus memoryDevice, a uint16) error {
	c.setFlag(flagCarry)
	return nil
}

func sed(c *cpu, bus memoryDevice, a uint16) error {
	c.setFlag(flagDecimal)
	return nil
}

func sei(c *cpu, bus memoryDevice, a uint16) error {
	c.setFlag(flagDisableIRQ)
	return nil
}

func sta(c *cpu, bus memoryDevice, a uint16) error {
	return bus.write(a, c.a)
}

func stx(c *cpu, bus memoryDevice, a uint16) error {
	return bus.write(a, c.x)
}

func sty(c *cpu, bus memoryDevice, a uint16) error {
	return bus.write(a, c.y)
}

func tax(c *cpu, bus memoryDevice, a uint16) error {
	c.setSignFlag(c.a)
	c.setZeroFlag(c.a)
	c.x = c.a
	return nil
}

func tay(c *cpu, bus memoryDevice, a uint16) error {
	c.setSignFlag(c.a)
	c.setZeroFlag(c.a)
	c.y = c.a
	return nil
}

func tsx(c *cpu, bus memoryDevice, a uint16) error {
	c.setSignFlag(c.sp)
	c.setZeroFlag(c.sp)
	c.x = c.sp
	return nil
}

func txa(c *cpu, bus memoryDevice, a uint16) error {
	c.setSignFlag(c.x)
	c.setZeroFlag(c.x)
	c.a = c.x
	return nil
}

func txs(c *cpu, bus memoryDevice, a uint16) error {
	c.sp = c.x
	return nil
}

func tya(c *cpu, bus memoryDevice, a uint16) error {
	c.setSignFlag(c.y)
	c.setZeroFlag(c.y)
	c.a = c.y
	return nil
}

func branch(c *cpu, a uint16) {
	c.clock++
	if pageCrossed(c.pc, a) {
		c.clock++
		c.pc = a
	}
}
