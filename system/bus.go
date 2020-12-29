package system

import "errors"

// CPU bus addresses
const (
	// wram mapper addresses
	wramLowAddr  uint16 = 0x0000
	wramHighAddr uint16 = 0x1fff
	wramMirror   uint16 = 0x800

	// ppu registers addresses
	ppuRegistersLowAddr  uint16 = 0x2000
	ppuRegistersHighAddr uint16 = 0x3fff
	ppuRegistersMirror   uint16 = 0x8

	// mapped device registers
	ppuOAMAddr uint16 = 0x4007

	// cartridge address range
	cartridgeLowAddr  uint16 = 0x4020
	cartridgeHighAddr uint16 = 0xffff
)

// PPU bus addresses
const (
	ppuBusMirror = 0x4000

	// pattern table addresses
	patternTablesLowAddr  uint16 = 0x0000
	patternTablesHighAddr uint16 = 0x1fff

	// vram addresses (name tables)
	vramLowAddr  uint16 = 0x2000
	vramHighAddr uint16 = 0x3eff
	vramSize     uint16 = 0x800

	// palette ram addresses
	paletteLowAddr  uint16 = 0x3ff0
	paletteHighAddr uint16 = 0x3fff
	paletteMirror   uint16 = 0x20
)

// cpuBus handles all memory accesses from the CPU.
// Address Space:
// 0x0000 - 0x1fff : WRAM (mirrored 0x800)
// 0x2000 - 0x3fff : PPU registers (mirrored 0x8)
// 0x4000 - 0x4017 : APU and IO registers
// 0x4018 - 0x401f : Test mode features (ignored)
// 0x4020 - 0xffff : Cartridge (PRG ROM, PRG RAM, and mappers)
type cpuBus struct {
	wram      [wramMirror]uint8
	ppu       *ppu
	cartridge *cartridge
}

func (b *cpuBus) write(a uint16, v uint8) error {
	switch {
	case a <= wramHighAddr:
		i := mirrorIndex(a, wramLowAddr, wramMirror)
		b.wram[i] = v
	case a <= ppuRegistersHighAddr:
		i := mirrorIndex(a, ppuRegistersLowAddr, ppuRegistersMirror)
		return b.ppu.write(i, v)
	case a == ppuOAMAddr:
		return b.ppu.oamDMA(v, b)
	case a >= cartridgeLowAddr && a <= cartridgeHighAddr:
		// cartridge access
	default:
		// TODO: test features and APU
	}
	return nil
}

func (b *cpuBus) read(a uint16) (uint8, error) {
	switch {
	case a <= wramHighAddr:
		i := mirrorIndex(a, wramLowAddr, wramMirror)
		return b.wram[i], nil
	case a <= ppuRegistersHighAddr:
		i := mirrorIndex(a, ppuRegistersLowAddr, ppuRegistersMirror)
		return b.ppu.read(i)
	case a >= cartridgeLowAddr && a <= cartridgeHighAddr:
		// cartridge access
	default:
		// TODO: test features and APU
	}
	return 0, nil
}

// ppuBus handles all memory accesses from the ppu.
// Address Space:
// 0x0000 - 0x1fff - pattern tables (mapped to CHR)
// 0x2000 - 0x2fff - vram (name tables)
// 0x3000 - 0x3eff - mirror of 0x2000 - 0x2eff
// 0x3f00 - 0x3fff - palette control
type ppuBus struct {
	cartridge  cartridge
	vram       [vramSize]uint8 // used for name tables
	paletteRAM [paletteMirror]uint8
}

func (b *ppuBus) write(a uint16, v uint8) error {
	a %= ppuBusMirror

	switch {
	case a <= patternTablesHighAddr:
		return b.cartridge.writeCHR(a, v)
	case a <= vramHighAddr:
		m := b.cartridge.vramMirror()
		i := m.index(a, vramLowAddr, nameTableSize)
		b.vram[i] = v
	case a <= paletteHighAddr:
		i := mirrorIndex(a, paletteLowAddr, paletteMirror)
		b.paletteRAM[i] = v
	}
	return nil
}

func (b *ppuBus) read(a uint16) (uint8, error) {
	a %= ppuBusMirror

	switch {
	case a <= patternTablesHighAddr:
		return b.cartridge.readCHR(a)
	case a <= vramHighAddr:
		m := b.cartridge.vramMirror()
		i := m.index(a, vramLowAddr, nameTableSize)
		return b.vram[i], nil
	case a <= paletteHighAddr:
		i := mirrorIndex(a, paletteLowAddr, paletteMirror)
		return b.paletteRAM[i], nil
	}
	return 0, errors.New("oob PPU bus read")
}
