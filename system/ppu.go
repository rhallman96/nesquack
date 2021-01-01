package system

import (
	"errors"
)

const (
	DrawWidth  = 256
	DrawHeight = 240

	oamSize              = 0x100
	nameTableSize uint16 = 0x400

	scanlineCount  = 261
	dotCount       = 341
	postRenderLine = 240

	// vram down increment for data accesses
	vramDown = 32

	// 1 CPU cycle = 3 PPU cycles
	ppuCycleRatio = 3
)

type ppu struct {
	drawer Drawer

	bus *ppuBus

	dot, scanline int

	oamAddr uint8
	oam     [oamSize]uint8

	scrollX, scrollY uint8
	scrollHi         bool

	address   uint16
	addressHi bool

	nameTableBaseAddr  uint16
	spriteTableAddr    uint16
	bgPatternTableAddr uint16

	// render flags
	largeSprites        bool
	vblankNMI           bool
	vramDownInc         bool
	grayscale           bool
	showBgLeft          bool
	showSpritesLeft     bool
	showBg              bool
	showSprites         bool
	eRed, eGreen, eBlue bool
	spriteZeroHit       bool
	vBlankPeriod        bool
	spriteOverflow      bool

	// last value written to a PPU register, used for status read
	lastWrite uint8
}

func newPPU(drawer Drawer, bus *ppuBus) *ppu {
	return &ppu{
		drawer: drawer,
		bus:    bus,
	}
}

func (p *ppu) step(cpuCycles uint64) {
	for i := uint64(0); i < cpuCycles; i++ {
		p.drawer.DrawPixel(p.dot, p.scanline, palette[(p.dot+p.scanline)%len(palette)])
		p.dot++
		if p.dot == DrawWidth {
			p.dot = 0
			p.scanline++
			if p.scanline == DrawHeight {
				p.scanline = 0
				p.drawer.CompleteFrame()
			}
		}
	}
}

func (p *ppu) write(a uint16, v uint8) error {
	switch a {
	case 0:
		p.writeCtrl(v)
	case 1:
		p.writeMask(v)
	case 3:
		p.writeOamAddr(v)
	case 4:
		p.writeOamData(v)
	case 5:
		p.writeScroll(v)
	case 6:
		p.writeAddress(v)
	case 7:
		return p.writeData(v)
	default:
		return errors.New("illegal ppu write")
	}
	return nil
}

func (p *ppu) read(a uint16) (uint8, error) {
	switch a {
	case 2:
		return p.readStatus()
	case 4:
		return p.readOamData()
	case 7:
		return p.readData()
	default:
		return 0, errors.New("illegal ppu read")
	}
}

func (p *ppu) writeCtrl(v uint8) {
	p.nameTableBaseAddr = vramLowAddr + (uint16(v&0x03) * nameTableSize)
	p.vramDownInc = isBitSet(v, 2)

	if isBitSet(v, 3) {
		p.spriteTableAddr = 0x1000
	} else {
		p.spriteTableAddr = 0
	}

	if isBitSet(v, 4) {
		p.bgPatternTableAddr = 0x1000
	} else {
		p.bgPatternTableAddr = 0
	}

	p.largeSprites = isBitSet(v, 5)
	p.vblankNMI = isBitSet(v, 7)

	p.lastWrite = v
}

func (p *ppu) writeMask(v uint8) {
	p.grayscale = isBitSet(v, 0)
	p.showBgLeft = isBitSet(v, 1)
	p.showSpritesLeft = isBitSet(v, 2)
	p.showBg = isBitSet(v, 3)
	p.showSprites = isBitSet(v, 4)
	p.eRed = isBitSet(v, 5)
	p.eGreen = isBitSet(v, 6)
	p.eBlue = isBitSet(v, 7)

	p.lastWrite = v
}

func (p *ppu) readStatus() (uint8, error) {
	var r uint8 = 0

	// bottom 5 bits are last written value
	r |= (p.lastWrite & 0x1f)

	if p.vBlankPeriod {
		r |= (1 << 7)
	}

	if p.spriteZeroHit {
		r |= (1 << 6)
	}

	if p.spriteOverflow {
		r |= (1 << 5)
	}

	return r, nil
}

func (p *ppu) writeOamAddr(v uint8) {
	p.lastWrite = v
	p.oamAddr = v
}

func (p *ppu) readOamData() (uint8, error) {
	return p.oam[p.oamAddr], nil
}

func (p *ppu) writeOamData(v uint8) {
	p.lastWrite = v
	p.oam[p.oamAddr] = v
	p.oamAddr++
}

func (p *ppu) writeScroll(v uint8) {
	p.lastWrite = v
	if p.scrollHi {
		p.scrollX = v
	} else {
		p.scrollY = v
	}
	p.scrollHi = !p.scrollHi
}

func (p *ppu) writeAddress(v uint8) {
	p.lastWrite = v
	if p.addressHi {
		p.address &= 0x00ff
		p.address |= (uint16(v) << 8)
	} else {
		p.address = uint16(v)
	}
	p.addressHi = !p.addressHi
}

func (p *ppu) readData() (uint8, error) {
	r, err := p.bus.read(p.address)
	if err != nil {
		return 0, err
	}

	if p.vramDownInc {
		p.address += vramDown
	} else {
		p.address++
	}

	return r, nil
}

func (p *ppu) writeData(v uint8) error {
	p.lastWrite = v
	err := p.bus.write(p.address, v)
	if err != nil {
		return err
	}

	if p.vramDownInc {
		p.address += vramDown
	} else {
		p.address++
	}

	return nil
}

func (p *ppu) oamDMA(v uint8, bus *cpuBus) error {
	p.lastWrite = v
	addr := uint16(v) << 8
	var i uint16
	for i = 0; i < oamSize; i++ {
		r, err := bus.read(addr + i)
		if err != nil {
			return err
		}
		p.oam[i] = r
	}
	return nil
}
