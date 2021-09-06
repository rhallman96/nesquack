package system

type addressMode func(c *cpu, bus memoryDevice, ac bool) (uint16, error)

func accumulator(c *cpu, bus memoryDevice, ac bool) (uint16, error) {
	return 0, nil
}

func absolute(c *cpu, bus memoryDevice, ac bool) (uint16, error) {
	a, err := readWord(bus, c.pc)
	if err != nil {
		return 0, err
	}
	c.pc += 2
	return a, nil
}

func absoluteX(c *cpu, bus memoryDevice, ac bool) (uint16, error) {
	a, err := readWord(bus, c.pc)
	if err != nil {
		return 0, err
	}
	c.pc += 2
	result := a + uint16(c.x)
	if ac && result&0xff00 != a&0xff00 {
		c.clock++
	}
	return result, nil
}

func absoluteY(c *cpu, bus memoryDevice, ac bool) (uint16, error) {
	a, err := readWord(bus, c.pc)
	if err != nil {
		return 0, err
	}
	c.pc += 2
	result := a + uint16(c.y)
	if ac && result&0xff00 != a&0xff00 {
		c.clock++
	}
	return result, nil
}

func immediate(c *cpu, bus memoryDevice, ac bool) (uint16, error) {
	a := c.pc
	c.pc++
	return a, nil
}

func implied(c *cpu, bus memoryDevice, ac bool) (uint16, error) {
	return 0, nil
}

func indirect(c *cpu, bus memoryDevice, ac bool) (uint16, error) {
	a, err := readWord(bus, c.pc)
	if err != nil {
		return 0, err
	}
	c.pc += 2
	a, err = readWordPage(bus, a)
	if err != nil {
		return 0, err
	}
	return a, nil
}

func indirectX(c *cpu, bus memoryDevice, ac bool) (uint16, error) {
	v, err := bus.read(c.pc)
	if err != nil {
		return 0, err
	}
	c.pc++
	a, err := readWordZeroPage(bus, v+c.x)
	if err != nil {
		return 0, err
	}
	return a, nil
}

func indirectY(c *cpu, bus memoryDevice, ac bool) (uint16, error) {
	v, err := bus.read(c.pc)
	if err != nil {
		return 0, err
	}
	c.pc++
	a, err := readWordZeroPage(bus, v)
	if err != nil {
		return 0, err
	}
	result := a + uint16(c.y)
	if ac && a&0xff00 != result&0xff00 {
		c.clock++
	}
	return result, nil
}

func relative(c *cpu, bus memoryDevice, ac bool) (uint16, error) {
	v, err := bus.read(c.pc)
	if err != nil {
		return 0, err
	}
	c.pc++
	return c.pc + uint16(int8(v)), nil
}

func zeroPage(c *cpu, bus memoryDevice, ac bool) (uint16, error) {
	v, err := bus.read(c.pc)
	if err != nil {
		return 0, err
	}
	c.pc++
	return uint16(v), nil
}

func zeroPageX(c *cpu, bus memoryDevice, ac bool) (uint16, error) {
	v, err := bus.read(c.pc)
	if err != nil {
		return 0, err
	}
	c.pc++
	return uint16(v + c.x), nil
}

func zeroPageY(c *cpu, bus memoryDevice, ac bool) (uint16, error) {
	v, err := bus.read(c.pc)
	if err != nil {
		return 0, err
	}
	c.pc++
	return uint16(v + c.y), nil
}
