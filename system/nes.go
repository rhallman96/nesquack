package system

type NES struct {
	cpu *cpu
}

func NewNES() *NES {
	return &NES{}
}
