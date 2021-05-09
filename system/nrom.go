package system

import (
	"errors"
	"fmt"
)

// nrom CPU banks
// 0x6000 - 0x7fff: prg RAM, mirrored if necessary
// 0x8000 - 0xbfff: first bank of prg ROM
// 0xc000 - 0xffff: second bank of prg ROM (or mirror of first bank)
type nrom struct {
	prgROM []uint8
	prgRAM []uint8
	chr    []uint8

	mirror mirrorMode
}

func (c *nrom) read(a uint16) (uint8, error) {
	switch {
	case (a >= prgRAMLowAddr) && (a <= prgRAMHighAddr):
		i := mirrorIndex(a, 0x6000, uint16(len(c.prgRAM)))
		return c.prgRAM[i], nil
	case a >= prgROMLowAddr:
		i := mirrorIndex(a, prgROMLowAddr, uint16(len(c.prgROM)))
		return c.prgROM[i], nil
	default:
		return 0, errors.New(fmt.Sprintf("oob nrom read at 0x%x", a))
	}
}

func (c *nrom) write(a uint16, v uint8) error {
	switch {
	case (a >= prgRAMLowAddr) && (a <= prgRAMHighAddr):
		i := mirrorIndex(a, 0x6000, uint16(len(c.prgRAM)))
		c.prgRAM[i] = v
	default:
		return errors.New(fmt.Sprintf("oob nrom write at 0x%x", a))
	}
	return nil
}

func (c *nrom) readCHR(a uint16) (uint8, error) {
	if a >= uint16(len(c.chr)) {
		return 0, errors.New("oob CHR read")
	}
	return c.chr[a], nil
}

func (c *nrom) writeCHR(a uint16, v uint8) error {
	if a >= uint16(len(c.chr)) {
		return errors.New("oob CHR write")
	}
	c.chr[a] = v
	return nil
}

func (c *nrom) vramMirror() mirrorMode {
	return c.mirror
}
