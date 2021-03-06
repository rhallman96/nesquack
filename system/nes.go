package system

// NES represents the system at its highest level.
type NES interface {
	// Step executes a single instruction within the NES CPU.
	Step() error
}

type nes struct {
	cpu *cpu
	ppu *ppu
}

// NewNES constructs a new NES
func NewNES(rom []uint8, drawer Drawer, c1 Controller) (NES, error) {
	cartridge, err := createCartridge(rom)
	if err != nil {
		return nil, err
	}

	// hook up controllers
	j1 := &joypad{controller: c1}

	// create system buses
	ppuBus := newPPUBus(cartridge)
	ppu := newPPU(drawer, ppuBus)

	cpuBus := newCPUBus(ppu, cartridge, j1)
	cpu, err := newCPU(cpuBus)
	ppu.cpu = cpu

	if err != nil {
		return nil, err
	}

	return &nes{
		cpu: cpu,
		ppu: ppu,
	}, nil
}

// Step fetches and executes one instruction on the NES's CPU.
// The PPU, MMU and all other peripherals will be updated accordingly.
func (n *nes) Step() error {
	prevCycles := n.cpu.clock
	err := n.cpu.step()
	if err != nil {
		return err
	}
	cycles := n.cpu.clock - prevCycles

	err = n.ppu.step(cycles)
	if err != nil {
		return err
	}
	return nil
}
