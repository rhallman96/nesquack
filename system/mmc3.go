package system

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
	bankRegs     []uint8

	mirror mirrorMode
}

func (c *mmc3) read(a uint16) (uint8, error) {
	return 0, nil
}

func (c *mmc3) write(a uint16, v uint8) error {
	return nil
}
