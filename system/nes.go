package system

type NES struct {
	cpu *cpu
	ppu *ppu

	cpuBus *cpuBus
	ppuBus *ppuBus
}

func NewNES() *NES {
	return &NES{}
}

func (n *NES) Step() error {
	err := n.cpu.step()
	if err != nil {
		return err
	}
	return nil
}
