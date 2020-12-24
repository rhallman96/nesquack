package system

import (
	"errors"
)

// memoryDevice represents any memory-mapped device in the NES, including
// peripherals, the PPU, and system buses.
type memoryDevice interface {
	write(a uint16, v uint8) error
	read(a uint16) (uint8, error)
}

func writeWord(d memoryDevice, a uint16, v uint16) error {
	err := d.write(a, uint8(v&0xff))
	if err != nil {
		return err
	}
	return d.write(a+1, uint8(v>>8))
}

func readWord(d memoryDevice, a uint16) (uint16, error) {
	low, err := d.read(a)
	if err != nil {
		return 0, err
	}
	hi, err := d.read(a + 1)
	return (uint16(hi) << 8) + uint16(low), err
}

// readWordZeroPage reads a word whose address is specified at a given
// offset on the zero page. If the address is located at 0x00ff, the high
// byte will be read from 0x0000.
func readWordZeroPage(d memoryDevice, offset uint8) (uint16, error) {
	low, err := d.read(uint16(offset))
	if err != nil {
		return 0, err
	}
	hi, err := d.read(uint16(offset + 1))
	return (uint16(hi) << 8) + uint16(low), err
}

// mirrorIndex returns an array index for mirrored memory accesses
func mirrorIndex(a, start, mirror uint16) uint16 {
	return start + ((a - start) % mirror)
}

// pageCrossed indicates if two addresses are on different pages
func pageCrossed(p1, p2 uint16) bool {
	return (p1 & 0xff00) != (p2 & 0xff00)
}

const (
	wramLowAddr  uint16 = 0x0000
	wramHighAddr uint16 = 0x2000
	wramMirror   uint16 = 0x800
)

// Working RAM on the NES. Occupies the address space 0x0000 - 0x1fff.
// wram is mirrored every 0x800 (2048) bytes, meaning that an access at
// 0x0801 is effectively the same as an access at 0x0001.
type wram struct {
	memory [wramMirror]uint8
}

func (w *wram) Write(a uint16, v uint8) error {
	if a >= wramHighAddr {
		return errors.New("oob wram write")
	}
	i := mirrorIndex(a, wramLowAddr, wramHighAddr)
	w.memory[i] = v
	return nil
}

func (w *wram) Read(a uint16) (uint8, error) {
	if a >= wramHighAddr {
		return 0, errors.New("oob wram read")
	}
	i := mirrorIndex(a, wramLowAddr, wramMirror)
	return w.memory[i], nil
}

// cpuBus handles all memory accesses from the CPU.
// Address Space:
// 0x0000 - 0x1fff : WRAM (mirrored 0x800)
// 0x2000 - 0x3fff : PPU registers (mirrored 0x8)
// 0x4000 - 0x4017 : APU and IO registers
// 0x4018 - 0x401f : Test mode features (ignored)
// 0x4020 - 0xffff : Cartridge (PRG ROM, PRG RAM, and mappers)
type cpuBus struct {
	wram *wram
}

func (b *cpuBus) Write(a uint16, v uint8) error {
	if a < wramHighAddr {
		return b.wram.Write(a, v)
	}
	return errors.New("oob CPU bus write")
}

func (b *cpuBus) Read(a uint16) (uint8, error) {
	if a < wramHighAddr {
		return b.wram.Read(a)
	}
	return 0, errors.New("oob CPU bus read")
}
