package system

import (
	"errors"
	"fmt"
	"log"
	"reflect"
)

var (
	inesPrefix = []uint8{0x4e, 0x45, 0x53, 0x1a}
)

const (
	prgROMSizeUnit = 0x4000 // 16 KB
	prgRAMSizeUnit = 0x2000 // 8 KB
	chrROMSizeUnit = 0x2000
	headerSize     = 0x10
	trainerSize    = 0x200

	prgRAMLowAddr  = 0x6000
	prgRAMHighAddr = 0x7fff
	prgROMLowAddr  = 0x8000
	prgROMHighAddr = 0xffff

	nromHeader = 0x00
)

// cartridge is a memory device with extended functionality for CHR accesses.
// It is defined as an interface because different cartridge designs contain
// defferent internal components and mappings that cannot be neatly described
// by a single struct.
type cartridge interface {
	memoryDevice
	readCHR(a uint16) (uint8, error)
	writeCHR(a uint16, v uint8) error
	vramMirror() mirrorMode
}

// createCartridge creates a cartridge based on the ROM's raw binary data.
// The cartridge header is assumed to be in the iNES format (NES 2.0 is not
// currently supported).
func createCartridge(rom []uint8) (cartridge, error) {
	// load iNES flags
	if !reflect.DeepEqual(rom[:4], inesPrefix) {
		return nil, errors.New("rom is not in iNES format")
	}
	prgROMSize := int(rom[4]) * prgROMSizeUnit
	chrROMSize := int(rom[5]) * chrROMSizeUnit
	mapper := (rom[7] & 0xf0) | (rom[6] >> 4)

	vMirror := isBitSet(rom[6], 0)
	// hasBattery := isBitSet(data[6], 1)
	hasTrainer := isBitSet(rom[6], 2)
	ignoreMirror := isBitSet(rom[6], 3)

	var ciMirror mirrorMode = onePage
	if !ignoreMirror {
		if vMirror {
			ciMirror = vertical
		} else {
			ciMirror = horizontal
		}
	}

	// initialize prgROM, prgRAM, and CHR
	prgRAMSize := int(rom[8]) * prgRAMSizeUnit
	if prgRAMSize == 0 {
		prgRAMSize = prgRAMSizeUnit
	}

	prgROMIndex := headerSize
	if hasTrainer {
		prgROMIndex += trainerSize
	}
	chrROMIndex := prgROMIndex + prgROMSize
	prgROM := rom[prgROMIndex : prgROMIndex+prgROMSize]
	prgRAM := make([]uint8, prgRAMSize, prgRAMSize)
	chrROM := rom[chrROMIndex : chrROMIndex+chrROMSize]

	log.Printf("PRG ROM: %d bytes", len(prgROM))
	log.Printf("PRG RAM: %d bytes", len(prgRAM))
	log.Printf("CHR: %d bytes", len(chrROM))

	// create a cartridge corresponding to iNES metadata
	var c cartridge

	switch mapper {
	case nromHeader:
		c = &nrom{
			prgROM: prgROM,
			prgRAM: prgRAM,
			chr:    chrROM,
			mirror: ciMirror,
		}
	default:
		return nil, errors.New(fmt.Sprintf("unsupported iNES mapper 0x%x", mapper))
	}

	return c, nil
}
