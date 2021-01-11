package system

type NES struct {
	cpu *cpu
	ppu *ppu

	cpuBus *cpuBus
	ppuBus *ppuBus
}

func NewNES(rom []uint8, drawer Drawer) (*NES, error) {
	cartridge, err := createCartridge(rom)
	if err != nil {
		return nil, err
	}

	ppuBus := newPPUBus(cartridge)
	ppu := newPPU(drawer, ppuBus)

	cpuBus := newCPUBus(ppu, cartridge)
	cpu, err := newCPU(cpuBus)

	if err != nil {
		return nil, err
	}

	// TODO: investigate more elegant way of instantiating
	ppu.cpu = cpu

	return &NES{
		cpu: cpu,
		ppu: ppu,
	}, nil
}

func (n *NES) Step() error {
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
