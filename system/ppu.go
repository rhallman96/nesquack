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

	maxSprites        = 8
	smallSpriteHeight = 8
	largeSpriteHeight = 16

	spritePaletteAddr = 0x3f10

	// vram down increment for data accesses
	vramDown = 32

	// 1 CPU cycle = 3 PPU cycles
	ppuCycleRatio = 3

	// cycles taken by an OAM DMA transfer
	oamDMACycles = 514
)

type ppu struct {
	drawer Drawer

	bus *ppuBus
	cpu *cpu

	dot, scanline int
	frame         int

	oamAddr uint8
	oam     [oamSize]uint8

	scrollX, scrollY uint8
	address          uint16

	writeToggle bool

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

	// data read from vram is stored in a buffer
	dataReadBuffer uint8

	// whether or not a background tile was drawn at each dot
	bgPixelDrawn []bool
}

func newPPU(drawer Drawer, bus *ppuBus) *ppu {
	return &ppu{
		drawer:       drawer,
		bus:          bus,
		bgPixelDrawn: make([]bool, DrawWidth),
	}
}

func (p *ppu) step(cpuCycles uint64) error {
	cycles := cpuCycles * ppuCycleRatio

	for i := uint64(0); i < cycles; i++ {
		p.dot++

		// when rendering is enabled, the ppu skips an additional tick every odd frame
		if (p.showSprites || p.showBg) && (p.frame%2 == 1) &&
			(p.dot == dotCount-2) && (p.scanline == scanlineCount-1) {
			p.dot++
		}

		if p.dot == DrawWidth && (p.scanline < DrawHeight) {
			err := p.drawTiles()
			if err != nil {
				return err
			}
			err = p.drawSprites()
			if err != nil {
				return err
			}
		} else if p.dot == dotCount {
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
				p.frame++
			}
		}
	}
	return nil
}

func (p *ppu) drawTiles() error {
	if !p.showBg {
		return nil
	}

	bgColor, err := p.bus.read(paletteLowAddr)
	if err != nil {
		return err
	}

	for dot := 0; dot < DrawWidth; dot++ {
		p.bgPixelDrawn[dot] = false
		p.drawer.DrawPixel(dot, p.scanline, palette[bgColor])

		if !p.showBgLeft && dot < 8 {
			continue
		}

		// get tile value
		pixelX := dot + int(p.scrollX)
		pixelY := p.scanline + int(p.scrollY)
		tileX := pixelX / 8
		tileY := pixelY / 8

		tileIndex := (tileY % nameTableHeight) * nameTableWidth
		tileIndex += (tileX % nameTableWidth)
		tileIndex += ((tileX / nameTableWidth) * int(nameTableSize))
		tileIndex += ((tileY / nameTableHeight) * 2 * int(nameTableSize))

		tileValue, err := p.bus.read(p.nameTableBaseAddr + uint16(tileIndex))
		if err != nil {
			return err
		}

		// get tile pixel color
		patternAddr := p.bgPatternTableAddr + (uint16(tileValue) * 16) + uint16(pixelY%8)
		pValueLow, err := p.bus.read(patternAddr)
		if err != nil {
			return err
		}
		pValueHi, err := p.bus.read(patternAddr + 8)
		if err != nil {
			return err
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
			continue
		}

		p.bgPixelDrawn[dot] = true

		// get palette
		at := p.nameTableBaseAddr + uint16((tileIndex/nameTableGridSize)*nameTableSize)
		at += nameTableGridSize

		atIndexX := (tileX % nameTableWidth) / 4
		atIndexY := (tileY % nameTableHeight) / 4
		atAddr := at + uint16((atIndexY*8)+atIndexX)

		c, err := p.bus.read(atAddr)
		if err != nil {
			return err
		}

		if (tileY/2)%2 == 0 {
			if (tileX/2)%2 == 0 {
				// top left
				c = c & 0x3
			} else {
				// top right
				c = (c >> 2) & 0x3
			}
		} else {
			if (tileX/2)%2 == 0 {
				// bottom left
				c = (c >> 4) & 0x3
			} else {
				// bottom right
				c = (c >> 6) & 0x3
			}
		}

		var pIndex uint16 = paletteLowAddr + uint16(c*4) + uint16(cIndex)
		color, err := p.bus.read(pIndex)
		if err != nil {
			return err
		}
		p.drawer.DrawPixel(dot, p.scanline, palette[color])
	}

	return nil
}

func (p *ppu) drawSprites() error {
	p.spriteZeroHit = false

	if !p.showSprites {
		return nil
	}

	spriteHeight := smallSpriteHeight
	if p.largeSprites {
		spriteHeight = largeSpriteHeight
	}

	// find index of last sprite to draw
	/*
		lastSpriteIndex := oamSize - 4
		spriteCount := 0
		for i := 0; i < oamSize; i += 4 {
			y := int(p.oam[i]) - int(p.scrollY)
			if p.scanline >= y && (y < (p.scanline + spriteHeight)) {
				spriteCount++
				if spriteCount == maxSprites {
					lastSpriteIndex = i
					break
				}
			}
		}
	*/

	spriteCount := 0
	// iterate over sprites in reverse, since earlier sprites are drawn with higher priority
	for i := 0; i < oamSize; i += 4 {
		if spriteCount == maxSprites {
			break
		}
		y := int(p.oam[i]) + 1
		if (p.scanline < y) || (p.scanline >= y+spriteHeight) {
			continue
		}
		spriteCount++

		x := int(p.oam[i+3])
		if x < 0 || x >= DrawWidth {
			continue
		}

		inFront := isBitSet(p.oam[i+2], 5)
		pIndex := p.oam[i+2] & 0x3
		hFlip := isBitSet(p.oam[i+2], 6)
		vFlip := isBitSet(p.oam[i+2], 7)
		tileValue := p.oam[i+1]

		patternBaseAddr := p.spriteTableAddr
		if p.largeSprites {
			if isBitSet(p.oam[i+1], 0) {
				patternBaseAddr = 0x1000
			} else {
				patternBaseAddr = 0
			}
			tileValue &= 0xfe
		}

		// draw sprite
		yOffset := p.scanline - y
		if vFlip {
			yOffset = (yOffset - (yOffset % 8)) + (7 - (yOffset % 8))
		}

		patternAddr := patternBaseAddr + (uint16(tileValue) * 16) + uint16(yOffset)
		pValueLow, err := p.bus.read(patternAddr)
		if err != nil {
			return err
		}
		pValueHi, err := p.bus.read(patternAddr + 8)
		if err != nil {
			return err
		}

		for ix := 0; ix < 8; ix++ {
			xOffset := ix
			if hFlip {
				xOffset = 7 - ix
			}

			if x+xOffset >= DrawWidth {
				continue
			}

			var cIndex int = 0
			if isBitSet(pValueLow, uint8(7-(xOffset%8))) {
				cIndex |= 0x1
			}
			if isBitSet(pValueHi, uint8(7-(xOffset%8))) {
				cIndex |= 0x2
			}

			// pixels with a zero value are transparent
			if cIndex == 0 {
				continue
			}

			if (i == 0) && (p.bgPixelDrawn[x+xOffset]) {
				p.spriteZeroHit = true
			}

			var pIndex uint16 = spritePaletteAddr + uint16(pIndex*4) + uint16(cIndex)
			color, err := p.bus.read(pIndex)
			if err != nil {
				return err
			}

			if !(inFront && p.bgPixelDrawn[x+xOffset]) {
				p.drawer.DrawPixel(x+ix, p.scanline, palette[color])
			}
		}
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
		return nil
		//return errors.New(fmt.Sprintf("illegal ppu write at offset 0x%x", a))
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

	p.writeToggle = false

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
	if p.writeToggle {
		p.scrollY = v
	} else {
		p.scrollX = v
	}
	p.writeToggle = !p.writeToggle
}

func (p *ppu) writeAddress(v uint8) {
	p.lastWrite = v
	if p.writeToggle {
		p.address |= uint16(v)
	} else {
		p.address = (uint16(v) << 8)

		// This is a hack, described in this nesdev wiki thread:
		// http://forums.nesdev.com/viewtopic.php?f=3&t=5365
		if p.address == 0 {
			p.scrollX = 0
			p.scrollY = 0
			p.nameTableBaseAddr = 0x2000
		}
	}
	p.writeToggle = !p.writeToggle
}

func (p *ppu) readData() (uint8, error) {
	r, err := p.bus.read(p.address)
	if err != nil {
		return 0, err
	}

	// non-palette VRAM is buffered in a separate register, and reads are delayed by one
	result := r
	if p.address <= vramHighAddr {
		result = p.dataReadBuffer
		p.dataReadBuffer = r
	}

	if p.vramDownInc {
		p.address += vramDown
	} else {
		p.address++
	}

	return result, nil
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
	p.step(oamDMACycles)
	return nil
}
