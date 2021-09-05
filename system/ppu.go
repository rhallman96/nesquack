package system

import (
	"errors"
	"fmt"
)

const (
	DrawWidth  = 256
	DrawHeight = 240

	oamSize              = 0x100
	nameTableSize        = 0x400
	nameTableSectionSize = 4 * nameTableSize

	nameTableWidth    = 32
	nameTableHeight   = 30
	nameTableGridSize = nameTableWidth * nameTableHeight

	scanlineCount  = 262
	dotCount       = 341
	incScanlineDot = 260
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

	loopyX         uint8
	loopyT, loopyV uint16

	writeToggle bool

	spriteTableAddr    uint16
	bgPatternTableAddr uint16

	// render flags
	largeSprites        bool
	vblankNMI           bool
	vramDownInc         bool
	grayscale           bool
	showTilesLeft       bool
	showSpritesLeft     bool
	showTiles           bool
	showSprites         bool
	eRed, eGreen, eBlue bool
	spriteZeroHit       bool
	vBlankPeriod        bool
	spriteOverflow      bool

	// last value written to a PPU register, used for status read
	lastWrite uint8

	// data read from vram is stored in a buffer
	dataReadBuffer uint8

	// whether or not a pixel was drawn at each dot
	tilePixelDrawn   [DrawWidth]bool
	spritePixelDrawn [DrawWidth]bool
}

func newPPU(drawer Drawer, bus *ppuBus) *ppu {
	return &ppu{
		drawer: drawer,
		bus:    bus,
	}
}

func (p *ppu) step(cpuCycles uint64) error {
	cycles := cpuCycles * ppuCycleRatio

	for i := uint64(0); i < cycles; i++ {
		p.dot++
		if p.renderEnabled() {
			if p.scanline == scanlineCount-1 {
				// when rendering is enabled, the ppu skips an additional tick every odd frame
				if (p.frame%2 == 1) && (p.dot == dotCount-2) {
					p.dot++
				}
				// the y value of loopyT is copied on the last scanline repeatedly
				if p.dot >= 280 && p.dot <= 304 {
					p.copyScrollY()
				}
			}

			if (p.dot == incScanlineDot) && (p.scanline <= DrawHeight) {
				p.bus.cartridge.incScanline(p.cpu)
			}
		}

		if p.dot == DrawWidth && (p.scanline < DrawHeight) {
			// end of visible portion of scanline
			err := p.drawTiles()
			if err != nil {
				return err
			}
			err = p.drawSprites()
			if err != nil {
				return err
			}

			if p.renderEnabled() {
				p.incScrollY()
			}
		} else if p.dot == dotCount {
			// begin new scanline
			p.dot = 0
			p.scanline++

			switch {
			case ((p.scanline < DrawHeight) || (p.scanline == scanlineCount-1)):
				if p.renderEnabled() {
					p.copyScrollX()
				}
			case p.scanline == postRenderLine:
				p.drawer.CompleteFrame()
			case p.scanline == postRenderLine+1:
				p.vBlankPeriod = true
				if p.vblankNMI {
					p.cpu.triggerNMI()
				}
			case p.scanline == scanlineCount:
				p.vBlankPeriod = false
				p.scanline = 0
				p.frame++
			}
		}
	}

	return nil
}

func (p *ppu) renderEnabled() bool {
	return p.showTiles || p.showSprites
}

func (p *ppu) drawTiles() error {
	bgColor, err := p.bus.read(paletteLowAddr)
	if err != nil {
		return err
	}

	for dot := 0; dot < DrawWidth; dot++ {
		p.tilePixelDrawn[dot] = false
		p.spritePixelDrawn[dot] = false

		p.drawer.DrawPixel(dot, p.scanline, palette[bgColor])

		// only draw the background color if tile rendering is disabled
		if !p.showTiles {
			continue
		}

		// get tile value
		fineX := (int(p.loopyX) + dot) % 8
		pixelX := int((p.loopyV&0x1f)<<3) + fineX
		pixelY := int(p.scrollY())
		tileX := pixelX / 8
		tileY := pixelY / 8
		tileAddress := uint16(0x2000 | (p.loopyV & 0x0FFF))
		attributeAddr := 0x23C0 | (p.loopyV & 0x0c00) | ((p.loopyV >> 4) & 0x38) | ((p.loopyV >> 2) & 0x07)

		// increment coarse X every 8 pixels (done after loopyV scroll value is fetched)
		if fineX == 7 {
			p.incCoarseX()
		}

		if p.dot < 8 && !p.showTilesLeft {
			fmt.Println("HERE")
			continue
		}

		// get tile pixel color
		tileValue, err := p.bus.read(tileAddress)
		if err != nil {
			return err
		}
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

		p.tilePixelDrawn[dot] = true

		// get tile palette
		c, err := p.bus.read(attributeAddr)
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

	spriteCount := 0
	// iterate over sprites in reverse, since earlier sprites are drawn with higher priority
	for i := 0; i <= oamSize-4; i += 4 {
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

		inFront := !isBitSet(p.oam[i+2], 5)
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
			yOffset = (yOffset - (yOffset % spriteHeight)) + (spriteHeight - 1 - (yOffset % spriteHeight))
		}

		patternAddr := patternBaseAddr + ((uint16(tileValue) + (uint16(yOffset) / 8)) * 16) + uint16(yOffset%8)
		pValueLow, err := p.bus.read(patternAddr)
		if err != nil {
			return err
		}
		pValueHi, err := p.bus.read(patternAddr + 8)
		if err != nil {
			return err
		}

		for ix := 0; ix < 8; ix++ {
			if x+ix >= DrawWidth {
				break
			}

			xOffset := ix
			if hFlip {
				xOffset = 7 - ix
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

			if (i == 0) && (p.tilePixelDrawn[x+ix]) {
				p.spriteZeroHit = true
			} else if p.spritePixelDrawn[x+ix] {
				continue
			}
			p.spritePixelDrawn[x+ix] = true

			var pIndex uint16 = spritePaletteAddr + uint16(pIndex*4) + uint16(cIndex)
			color, err := p.bus.read(pIndex)
			if err != nil {
				return err
			}

			if inFront || !p.tilePixelDrawn[x+ix] {
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
	p.writeNameTableBaseAddr(v)
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
	p.showTilesLeft = isBitSet(v, 1)
	p.showSpritesLeft = isBitSet(v, 2)
	p.showTiles = isBitSet(v, 3)
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
		p.writeScrollY(v)
	} else {
		p.writeScrollX(v)
	}
	p.writeToggle = !p.writeToggle
}

func (p *ppu) writeAddress(v uint8) {
	p.lastWrite = v
	if p.writeToggle {
		p.loopyT &= 0xff00
		p.loopyT |= uint16(v)

		if p.loopyV&0x1000 == 0 && (p.loopyT&0x1000 != 0) {
			p.bus.cartridge.incScanline(p.cpu)
		}

		p.loopyV = p.loopyT
	} else {
		p.loopyT &= 0xff
		p.loopyT |= (uint16(v&0x3f) << 8)
	}
	p.writeToggle = !p.writeToggle
}

func (p *ppu) readData() (uint8, error) {
	r, err := p.bus.read(p.loopyV)
	if err != nil {
		return 0, err
	}

	// non-palette VRAM is buffered, and reads are subsequently delayed by one
	result := r
	if p.loopyV <= vramHighAddr {
		result = p.dataReadBuffer
		p.dataReadBuffer = r
	} else if p.loopyV <= paletteHighAddr {
		p.dataReadBuffer, err = p.bus.read(p.loopyV - 0x1000)
		if err != nil {
			return 0, err
		}
	}

	if p.vramDownInc {
		p.loopyV += vramDown
	} else {
		p.loopyV++
	}

	return result, nil
}

func (p *ppu) writeData(v uint8) error {
	p.lastWrite = v
	err := p.bus.write(p.loopyV, v)
	if err != nil {
		return err
	}

	if p.vramDownInc {
		p.loopyV += vramDown
	} else {
		p.loopyV++
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
		p.oam[p.oamAddr+uint8(i)] = r
	}

	p.step(oamDMACycles)
	return nil
}

func (p *ppu) writeScrollX(v uint8) {
	p.loopyT &= 0xffe0
	p.loopyT |= uint16(v >> 3)
	p.loopyX = v & 0x7
}

func (p *ppu) scrollX() uint8 {
	return (uint8(p.loopyV&0x1f) << 3) | p.loopyX
}

func (p *ppu) writeScrollY(v uint8) {
	p.loopyT &= 0x8fff
	p.loopyT |= (uint16(v&0x7) << 12)
	p.loopyT &= 0xfc1f
	p.loopyT |= (uint16(v&0xf8) << 2)
}

func (p *ppu) scrollY() uint8 {
	r := (p.loopyV >> 2) & 0xf8
	r |= ((p.loopyV >> 12) & 0x7)
	return uint8(r)
}

func (p *ppu) writeNameTableBaseAddr(v uint8) {
	p.loopyT &= 0xf3ff
	p.loopyT |= (uint16(v&0x3) << 10)
}

func (p *ppu) nameTableBaseAddr() uint16 {
	return vramLowAddr + (((p.loopyV >> 10) & 0x3) * nameTableSize)
}

// copyScrollX copies the x scroll contents of the temp register into the v register.
// This is called at dot 257 of every scanline.
func (p *ppu) copyScrollX() {
	p.loopyV = (p.loopyV & 0xfbe0) | (p.loopyT & 0x041f)
}

// copyScrollY copies the y scroll contents of the temp register into the v register.
// This is called between dots 280 and 304 of the pre-render scanline.
func (p *ppu) copyScrollY() {
	p.loopyV = (p.loopyV & 0x041f) | (p.loopyT & 0xfbe0)
}

// incCoarseX increments the coarse x register for each tile when a scanline is
// rendered. (i.e., once every 8 pixels)
func (p *ppu) incCoarseX() {
	if (p.loopyV & 0x001f) == 31 {
		p.loopyV &= 0xffe0
		p.loopyV ^= 0x0400
	} else {
		p.loopyV += 1
	}
}

// incScrollY increments the scroll y coordinate. This is called at dot 256 of each scanline.
func (p *ppu) incScrollY() {
	if (p.loopyV & 0x7000) != 0x7000 {
		p.loopyV += 0x1000
	} else {
		p.loopyV &= 0x8fff
		y := (p.loopyV & 0x03e0) >> 5
		if y == 29 {
			y = 0
			p.loopyV ^= 0x0800
		} else if y == 31 {
			y = 0
		} else {
			y++
		}
		p.loopyV = (p.loopyV & 0xfc1f) | (y << 5)
	}
}
