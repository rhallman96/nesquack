package system

type NES struct {
	cpu *cpu
	ppu *ppu

	cpuBus *cpuBus
	ppuBus *ppuBus
}

func NewNES(drawer Drawer) *NES {
	ppu := &ppu{drawer: drawer}
	return &NES{ppu: ppu}
}

func (n *NES) Step() error {
	return nil
}
`
