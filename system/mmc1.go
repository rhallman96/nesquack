package system

import (
	"errors"
	"fmt"
)

const (
	mmc1SRClearValue = 0x10

	controlRegisterHighAddr  = 0x9fff
	chrBank0RegisterHighAddr = 0xbfff
	chrBank1RegisterHighAddr = 0xdfff
	prgBankRegisterHighAddr  = 0xffff

	prgROMBankMode32K      = 0
	prgROMBankModeFixFirst = 2
	prgROMBankModeFixLast  = 3
)

// mmc1 CPU banks
// 0x6000 - 0x7fff: RAM (optional)
// 0x8000 - 0xbfff: first bank of prg ROM
// 0xc000 - 0xffff: switchable prg ROM bank

// mmc1 PPU banks
// 0x0000 - 0x0fff: switchable CHR bank
// 0x1000 - 0x1fff: switchable CHR bank
type mmc1 struct {
	prgROM []uint8
	prgRAM []uint8
	chr    []uint8

	prgROMBank              int
	chrLowBank, chrHighBank int
	chrBank8K               bool

	prgROMBankMode int

	prgRAMEnabled bool
	mirror        mirrorMode
	sr            uint8
}

func (c *mmc1) read(a uint16) (uint8, error) {
	switch {
	case (a >= prgRAMLowAddr) && (a <= prgRAMHighAddr):
		if !c.prgRAMEnabled {
			return 0, nil
		}
		i := mirrorIndex(a, 0x6000, uint16(len(c.prgRAM)))
		return c.prgRAM[i], nil
	case a >= prgROMLowAddr:
		return c.readPRG(a)
	default:
		return 0, errors.New(fmt.Sprintf("oob mmc1 read at 0x%x", a))
	}
	return 0, nil
}

func (c *mmc1) write(a uint16, v uint8) error {
	switch {
	case (a >= prgRAMLowAddr) && (a <= prgRAMHighAddr):
		if c.prgRAMEnabled {
			i := mirrorIndex(a, 0x6000, uint16(len(c.prgRAM)))
			c.prgRAM[i] = v
		}
	case a >= prgROMLowAddr:
		c.writeShiftRegister(a, v)
	default:
		return errors.New(fmt.Sprintf("oob mmc1 write at 0x%x", a))
	}
	return nil
}

func (c *mmc1) readCHR(a uint16) (uint8, error) {
	if a >= uint16(len(c.chr)) {
		return 0, errors.New("oob CHR read")
	}
	return c.chr[c.getCHRIndex(a)], nil
}

func (c *mmc1) writeCHR(a uint16, v uint8) error {
	if a >= uint16(len(c.chr)) {
		return errors.New("oob CHR write")
	}
	c.chr[c.getCHRIndex(a)] = v
	return nil
}

func (c *mmc1) readPRG(a uint16) (uint8, error) {
	return c.prgROM[c.getPRGAddress(a)], nil
}

func (c *mmc1) vramMirror() mirrorMode {
	return c.mirror
}

func (c *mmc1) writeShiftRegister(a uint16, v uint8) {
	// TODO: ignore successive writes for improved compatibility

	if v >= 0x80 {
		// if bit 7 is set, we revert to the default value
		c.sr = mmc1SRClearValue
		c.prgROMBankMode = prgROMBankModeFixLast
		return
	}

	next := c.sr >> 1
	if v&0x1 != 0 {
		next |= mmc1SRClearValue
	}

	if c.sr&0x1 == 0 {
		c.sr = next
		return
	}

	switch {
	case a <= controlRegisterHighAddr:
		c.writeControlRegister(next)
	case a <= chrBank0RegisterHighAddr:
		c.writeCHRLowBankRegister(next)
	case a <= chrBank1RegisterHighAddr:
		c.writeCHRHighBankRegister(next)
	case a <= prgBankRegisterHighAddr:
		c.writePRGBankRegister(next)
	}

	c.sr = mmc1SRClearValue
}

func (c *mmc1) writeControlRegister(v uint8) {
	c.chrBank8K = !isBitSet(v, 4)
	c.prgROMBankMode = int((v >> 2) & 0x3)

	m := v & 0x3
	switch m {
	case 0:
		c.mirror = onePage
	case 1:
		c.mirror = onePageHigh
	case 2:
		c.mirror = vertical
	case 3:
		c.mirror = horizontal
	}
}

func (c *mmc1) writeCHRLowBankRegister(v uint8) {
	c.chrLowBank = int(v & 0x1f)
}

func (c *mmc1) writeCHRHighBankRegister(v uint8) {
	c.chrHighBank = int(v & 0x1f)
}

func (c *mmc1) writePRGBankRegister(v uint8) {
	c.prgRAMEnabled = !isBitSet(v, 4)
	c.prgROMBank = int(v & 0xf)
}

func (c *mmc1) getPRGAddress(a uint16) int {
	prgAddr := int(a - prgROMLowAddr)
	switch c.prgROMBankMode {
	case prgROMBankModeFixFirst:
		if prgAddr < prgROMBankSize {
			return prgAddr
		}
		return (prgAddr % prgROMBankSize) + (c.prgROMBank * prgROMBankSize)
	case prgROMBankModeFixLast:
		if prgAddr >= prgROMBankSize {
			return len(c.prgROM) - (2 * prgROMBankSize) + prgAddr
		}
		return prgAddr + (c.prgROMBank * prgROMBankSize)
	default:
		bank := c.prgROMBank - (c.prgROMBank % 2)
		return prgAddr + (bank * prgROMBankSize)
	}
}

func (c *mmc1) getCHRIndex(a uint16) int {
	if c.chrBank8K {
		bank := c.chrLowBank - (c.chrLowBank % 2)
		return int(a) + (bank * chrBankSize)
	} else if a < chrBankSize {
		return int(a) + (c.chrLowBank * chrBankSize)
	}
	return int(a) - chrBankSize + (c.chrHighBank * chrBankSize)
}
