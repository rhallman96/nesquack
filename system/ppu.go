package system

import (
	"errors"
)

const (
	DrawWidth  = 256
	DrawHeight = 240

	oamSize       = 0x100
	nameTableSize = 0x400

	nameTableWidth    = 32
	nameTableHeight   = 30
	nameTableGridSize = nameTableWidth * nameTableHeight

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
	cpu *cpu

	dot, scanline int

	oamAddr uint8
	oam     [oamSize]uint8

	scrollX, scrollY uint8
	scrollLow        bool

	address    uint16
	addressLow bool

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

func (p *ppu) step(cpuCycles uint64) error {
	cycles := cpuCycles * ppuCycleRatio

	bgColor, _ := p.bus.read(paletteLowAddr)
	for i := uint64(0); i < cycles; i++ {

		if (p.dot < DrawWidth) && (p.scanline < DrawHeight) {
			p.drawer.DrawPixel(p.dot, p.scanline, palette[bgColor])

			bgDrawn, err := p.drawTiles()
			if err != nil {
				return err
			}
			p.drawSprites(bgDrawn)
		}

		// advance dot
		p.dot++
		if p.dot == dotCount {
			p.dot = 0
			p.scanline++
			switch p.scanline {
			case postRenderLine:
				p.drawer.CompleteFrame()
			case postRenderLine + 1:
				p.vBlankPeriod = true
				if p.vblankNMI {
					p.cpu.triggerNMI()
				}
			case scanlineCount:
				p.vBlankPeriod = false
				p.scanline = 0
			}
		}
	}
	return nil
}

func (p *ppu) drawTiles() (bool, error) {
	if !p.showBgLeft || !p.showBg {
		return false, nil
	}

	// get tile value
	pixelX := p.dot + int(p.scrollX)
	pixelY := p.scanline + int(p.scrollY)
	tileX := pixelX / 8
	tileY := pixelY / 8

	tileIndex := (tileY * nameTableWidth) + tileX
	tileIndex += ((tileX / nameTableWidth) * int(nameTableSize))
	tileIndex += ((tileY / nameTableHeight) * 2 * int(nameTableSize))

	tileValue, err := p.bus.read(p.nameTableBaseAddr + uint16(tileIndex))
	if err != nil {
		return false, err
	}

	// get tile pixel color
	patternAddr := p.bgPatternTableAddr + (uint16(tileValue) * 16) + uint16(pixelY%8)
	pValueLow, err := p.bus.read(patternAddr)
	if err != nil {
		return false, err
	}
	pValueHi, err := p.bus.read(patternAddr + 8)
	if err != nil {
		return false, err
	}

	var cIndex int = 0
	if isBitSet(pValueLow, uint8(7-(pixelX%8))) {
		cIndex |= 0x1
	}
	if isBitSet(pValueHi, uint8(7-(pixelX%8))) {
		cIndex |= 0x2
	}

	// pixels with a zero value are transparent
	if cIndex == 0 {
		return false, nil
	}

	// get palette
	at := p.nameTableBaseAddr + uint16((tileIndex/nameTableSize)*nameTableSize)
	at += nameTableGridSize

	atIndexX := (tileX % nameTableWidth) / 4
	atIndexY := (tileY % nameTableHeight) / 4
	atAddr := at + uint16((atIndexY*8)+atIndexX)

	c, err := p.bus.read(atAddr)
	if err != nil {
		return false, err
	}

	if (tileY/2)%2 == 0 {
		if (tileX/2)%2 == 0 {
			c = c & 0x3 // top left
		} else {
			c = (c >> 2) & 0x3
		}
	} else {
		if (tileX/2)%2 == 0 {
			c = (c >> 4) & 0x3
		} else {
			c = (c >> 6) & 0x3
		}
	}
	var pIndex uint16 = paletteLowAddr + uint16(c*4) + uint16(cIndex)
	color, err := p.bus.read(pIndex)
	if err != nil {
		return false, err
	}

	p.drawer.DrawPixel(p.dot, p.scanline, palette[color])
	return (c != 0), nil
}

func (p *ppu) drawSprites(bgDrawn bool) error {
	if !p.showSprites || !p.showSpritesLeft {
		return nil
	}
	return nil
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

	p.addressLow = false
	p.scrollLow = false

	// bottom 5 bits are last written value
	r |= (p.lastWrite & 0x1f)

	if p.vBlankPeriod {
		r |= (1 << 7)
		p.vBlankPeriod = false
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
	if p.scrollLow {
		p.scrollY = v
	} else {
		p.scrollX = v
	}
	p.scrollLow = !p.scrollLow
}

func (p *ppu) writeAddress(v uint8) {
	p.lastWrite = v
	if p.addressLow {
		p.address |= uint16(v)
	} else {
		p.address = (uint16(v) << 8)
	}
	p.addressLow = !p.addressLow
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
