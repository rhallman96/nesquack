package system

import (
	"errors"
	"fmt"
)

const (
	mmc3PRGRomBank1Low  = 0x8000
	mmc3PRGRomBank1High = 0x9fff
	mmc3PRGRomBank2Low  = 0xa000
	mmc3PRGRomBank2High = 0xbfff
	mmc3PRGRomBank3Low  = 0xc000
	mmc3PRGRomBank3High = 0xdfff
	mmc3PRGRomBank4Low  = 0xe000
	mmc3ROMBankSize     = 0x2000
	mmc3CHRBankSize     = 0x400
)

// mmc3 CPU banks
// 0x6000 - 0x7fff - switchable PRG RAM bank
// 0x8000 - 0x9fff - switchable PRG ROM bank
// 0xa000 - 0xbfff - switchable PRG ROM bank
// 0xc000 - 0xdfff - mirror of 0x8000 - 0x9fff
// 0xe000 - 0xffff - bank fixed to last PRG ROM bank

type mmc3 struct {
	prgROM []uint8
	prgRAM []uint8
	chr    []uint8

	bankRegIndex uint8
	bankRegs     [8]uint8

	mmcRegister    bool
	chrA12Inverted bool
	irqEnabled     bool

	// when set to true, triggers irq set to false
	triggerIrqDisable bool

	// scaline counter is not immediately reloaded
	triggerReload bool

	irqLatch, irqCounter uint8

	mirror mirrorMode
}

func (c *mmc3) read(a uint16) (uint8, error) {
	switch {
	case (a >= prgRAMLowAddr) && (a <= prgRAMHighAddr):
		return c.prgRAM[a-prgRAMLowAddr], nil
	case (a >= mmc3PRGRomBank1Low) && (a <= mmc3PRGRomBank1High):
		if c.mmcRegister {
			// low bank fixed to the second-last 8KB bank
			i := int(a-mmc3PRGRomBank1Low) + len(c.prgROM) - 0x4000
			return c.prgROM[i], nil
		}
		i := int(a-mmc3PRGRomBank1Low) + (int(c.bankRegs[6]) * mmc3ROMBankSize)
		return c.prgROM[i], nil
	case (a >= mmc3PRGRomBank2Low) && (a <= mmc3PRGRomBank2High):
		i := int(a-mmc3PRGRomBank2Low) + (int(c.bankRegs[7]) * mmc3ROMBankSize)
		return c.prgROM[i], nil
	case (a >= mmc3PRGRomBank3Low) && (a <= mmc3PRGRomBank3High):
		if c.mmcRegister {
			i := int(a-mmc3PRGRomBank3Low) + (int(c.bankRegs[6]) * mmc3ROMBankSize)
			return c.prgROM[i], nil
		}
		i := int(a-mmc3PRGRomBank3Low) + len(c.prgROM) - 0x4000
		return c.prgROM[i], nil
	case a >= mmc3PRGRomBank4Low:
		i := int(a-mmc3PRGRomBank4Low) + len(c.prgROM) - mmc3ROMBankSize
		return c.prgROM[i], nil
	}
	return 0, errors.New(fmt.Sprintf("oob mmc3 read at 0x%x", a))
}

func (c *mmc3) write(a uint16, v uint8) error {
	switch {
	case (a >= prgRAMLowAddr) && (a <= prgRAMHighAddr):
		c.prgRAM[a-prgRAMLowAddr] = v
	case (a >= prgROMLowAddr) && (a <= mmc3PRGRomBank1High):
		if a%2 == 0 {
			return c.writeBankSelectEven(v)
		}
		return c.writeBankSelectOdd(v)
	case (a > mmc3PRGRomBank1High) && (a <= mmc3PRGRomBank2High):
		if a%2 == 0 {
			return c.writeMirror(v)
		}
		// write protect intentionally not implemented
	case (a > mmc3PRGRomBank2High) && (a <= mmc3PRGRomBank3High):
		if a%2 == 0 {
			return c.writeIRQLatch(v)
		}
		return c.writeIRQReload(v)
	case a > mmc3PRGRomBank3High:
		if (a % 2) == 1 {
			c.irqEnabled = true
		} else {
			c.irqEnabled = false
			c.triggerIrqDisable = true
		}
	default:
		return errors.New(fmt.Sprintf("oob mmc3 write at 0x%x", a))
	}
	return nil
}

func (c *mmc3) readCHR(a uint16) (uint8, error) {
	if c.mmcRegister {
		return c.chr[c.getCHRIndexMMCRegisterOn(a)], nil
	}
	return c.chr[c.getCHRIndexMMCRegisterOff(a)], nil
}

func (c *mmc3) writeCHR(a uint16, v uint8) error {
	if c.mmcRegister {
		c.chr[c.getCHRIndexMMCRegisterOn(a)] = v
	} else {
		c.chr[c.getCHRIndexMMCRegisterOff(a)] = v
	}
	return nil
}

func (c *mmc3) vramMirror() mirrorMode {
	return c.mirror
}

func (c *mmc3) writeBankSelectEven(v uint8) error {
	c.bankRegIndex = v & 0x7
	c.mmcRegister = isBitSet(v, 6)
	c.chrA12Inverted = isBitSet(v, 7)
	return nil
}

func (c *mmc3) writeBankSelectOdd(v uint8) error {
	c.bankRegs[c.bankRegIndex] = v
	return nil
}

func (c *mmc3) writeMirror(v uint8) error {
	if isBitSet(v, 0) {
		c.mirror = horizontal
	} else {
		c.mirror = vertical
	}
	return nil
}

func (c *mmc3) writeIRQLatch(v uint8) error {
	c.irqLatch = v
	return nil
}

func (c *mmc3) writeIRQReload(v uint8) error {
	c.triggerReload = true
	return nil
}

func (c *mmc3) getCHRIndexMMCRegisterOff(a uint16) int {
	var bank uint8 = 0
	var offset uint16 = 0
	switch {
	case a < 0x400:
		offset = a
		bank = c.bankRegs[2]
	case a < 0x800:
		offset = a - 0x400
		bank = c.bankRegs[3]
	case a < 0xc00:
		offset = a - 0x800
		bank = c.bankRegs[4]
	case a < 0x1000:
		offset = a - 0xc00
		bank = c.bankRegs[5]
	case a < 0x1800:
		offset = a - 0x1000
		bank = c.bankRegs[0] & 0xfe
	default:
		offset = a - 0x1800
		bank = c.bankRegs[1] & 0xfe
	}

	return (int(bank) * mmc3CHRBankSize) + int(offset)
}

func (c *mmc3) getCHRIndexMMCRegisterOn(a uint16) int {
	var bank uint8 = 0
	var offset uint16 = 0
	switch {
	case a < 0x800:
		offset = a
		bank = c.bankRegs[0] & 0xfe
	case a < 0x1000:
		offset = a - 0x800
		bank = c.bankRegs[1] & 0xfe
	case a < 0x1400:
		offset = a - 0x1000
		bank = c.bankRegs[2]
	case a < 0x1800:
		offset = a - 0x1400
		bank = c.bankRegs[3]
	case a < 0x1c00:
		offset = a - 0x1800
		bank = c.bankRegs[4]
	default:
		offset = a - 0x1c00
		bank = c.bankRegs[5]
	}

	return (int(bank) * mmc3CHRBankSize) + int(offset)
}

func (c *mmc3) incScanline(cp *cpu) error {
	if c.triggerReload || c.irqCounter == 0 {
		c.irqCounter = c.irqLatch
		c.triggerReload = false
		cp.setIRQ(false)
	} else {
		c.irqCounter--
	}

	if c.irqCounter == 0 {
		cp.setIRQ(c.irqEnabled)
	}

	return nil
}
