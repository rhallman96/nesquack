package system

// mmc3 CPU banks
//
//
//
// mmc3 PPU banks
//
//
//

type mmc3 struct {
	prgROM []uint8
	prgRAM []uint8
	chr    []uint8

	mirror mirrorMode
}

func (c *mmc3) read(a uint16) (uint8, error) {
	return 0, nil
}

func (c *mmc3) write(a uint16, v uint8) error {
	return nil
}
