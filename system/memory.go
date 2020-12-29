package system

// memoryDevice represents any memory-mapped device in the NES, including
// peripherals, the PPU, and system buses.
type memoryDevice interface {
	write(a uint16, v uint8) error
	read(a uint16) (uint8, error)
}

func isBitSet(flag, bit uint8) bool {
	return (flag & (1 << bit)) != 0
}

func writeWord(d memoryDevice, a uint16, v uint16) error {
	err := d.write(a, uint8(v&0xff))
	if err != nil {
		return err
	}
	return d.write(a+1, uint8(v>>8))
}

func readWord(d memoryDevice, a uint16) (uint16, error) {
	low, err := d.read(a)
	if err != nil {
		return 0, err
	}
	hi, err := d.read(a + 1)
	return (uint16(hi) << 8) + uint16(low), err
}

// readWordZeroPage reads a word whose address is specified at a given
// offset on the zero page. If the address is located at 0x00ff, the high
// byte will be read from 0x0000.
func readWordZeroPage(d memoryDevice, offset uint8) (uint16, error) {
	low, err := d.read(uint16(offset))
	if err != nil {
		return 0, err
	}
	hi, err := d.read(uint16(offset + 1))
	return (uint16(hi) << 8) + uint16(low), err
}

// pageCrossed indicates if two addresses are on different pages
func pageCrossed(p1, p2 uint16) bool {
	return (p1 & 0xff00) != (p2 & 0xff00)
}
