package system

import (
	"errors"
	"fmt"
)

// cpu implements the Ricoh 2A03 (a MOS 6502 derivative) instruction set.
type cpu struct {
	bus *cpuBus

	pc             uint16
	a, x, y, p, sp uint8
	clock          uint64

	// nmi is falling edge sensitive
	irq, nmi bool
}

const (
	// status flags
	flagSign      uint8 = 1 << 7
	flagOverflow  uint8 = 1 << 6
	flagBreakHi   uint8 = 1 << 5
	flagBreak     uint8 = 1 << 4
	flagDecimal   uint8 = 1 << 3 // not supported, but may still be set
	flagInterrupt uint8 = 1 << 2
	flagZero      uint8 = 1 << 1
	flagCarry     uint8 = 1

	// boot up register values
	spStartValue     uint8 = 0xff
	statusStartValue uint8 = 0x34

	// system addresses
	spBaseAddress uint16 = 0x0100
	nmiVector            = 0xfffa
	resetVector          = 0xfffc
	irqVector            = 0xfffe

	// cpu cycles used for interrupt branching
	interruptCycles = 7
)

func newCPU(bus *cpuBus) (*cpu, error) {
	r := &cpu{
		bus: bus,
		sp:  spStartValue,
		p:   statusStartValue,
	}

	pc, err := readWord(bus, resetVector)
	if err != nil {
		return nil, err
	}
	r.pc = pc

	return r, nil
}

// step performs the next instruction in memory and returns how many cycles
// it took to execute.
func (c *cpu) step() error {
	// fetch the next opcode
	op, err := c.bus.read(c.pc)
	if err != nil {
		return err
	}

	// decode
	i := instructionSet[op]
	if i == nil {
		return errors.New(fmt.Sprintf("unknown op 0x%x at addr 0x%x", op, c.pc))
	}

	// execute
	err = i.execute(c)
	if err != nil {
		return err
	}

	// handle interrupts
	if c.nmi {
		c.clock += interruptCycles
		// indicates that the interrupt was not handled during a brk instruction
		c.setFlagValue(flagBreak, false)
		err = c.interrupt(c.bus, nmiVector)
		c.nmi = false
	} else if !c.isFlagSet(flagInterrupt) && c.irq {
		c.clock += interruptCycles
		c.setFlagValue(flagBreak, false)
		err = c.interrupt(c.bus, irqVector)
	}

	return err
}

// setIRQ sets the value of the IRQ line. True represents a low state, and false
// represents a high state.
func (c *cpu) setIRQ(v bool) {
	c.irq = v
}

// triggerNMI sets the NMI flag to true, which represents a level transition for
// the NMI line. The NMI flag is set to false as soon as the interrupt is handled
// during the next invocation of step.
func (c *cpu) triggerNMI() {
	c.nmi = true
}

func (c *cpu) interrupt(bus memoryDevice, v uint16) error {
	c.pc++
	err := c.pushWord(bus, c.pc)
	if err != nil {
		return err
	}
	c.setFlag(flagBreakHi)
	c.setFlag(flagInterrupt)
	c.push(bus, c.p)
	dest, err := readWord(bus, v)
	if err != nil {
		return err
	}
	c.pc = dest
	return nil
}

func (c *cpu) push(bus memoryDevice, v uint8) error {
	err := bus.write(spBaseAddress+uint16(c.sp), v)
	c.sp--
	return err
}

func (c *cpu) pull(bus memoryDevice) (uint8, error) {
	c.sp++
	v, err := bus.read(spBaseAddress + uint16(c.sp))
	return v, err
}

func (c *cpu) pushWord(bus memoryDevice, v uint16) error {
	err := c.push(bus, uint8(v>>8))
	if err != nil {
		return err
	}
	return c.push(bus, uint8(v))
}

func (c *cpu) pullWord(bus memoryDevice) (uint16, error) {
	low, err := c.pull(bus)
	if err != nil {
		return 0, err
	}
	hi, err := c.pull(bus)
	return (uint16(hi) << 8) + uint16(low), err
}

func (c *cpu) setFlag(flag uint8) {
	c.p |= flag
}

func (c *cpu) setFlagValue(flag uint8, v bool) {
	if v {
		c.p |= flag
	} else {
		c.p = c.p &^ flag
	}
}

func (c *cpu) setSignFlag(v uint8) {
	c.setFlagValue(flagSign, int8(v) < 0)
}

func (c *cpu) setZeroFlag(v uint8) {
	c.setFlagValue(flagZero, v == 0)
}

func (c *cpu) isFlagSet(flag uint8) bool {
	return (c.p & flag) != 0
}
