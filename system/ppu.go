package system

// CI RAM can be mirrored in three different configurations, depending
// on the wiring in a given cartridge
type mirror int

const (
	noMirror = iota
	horizontal
	vertical
)

// Drawer is an abstraction to draw pixel data to the screen.
// It is not implemented in this package and should instead
// be implemented with the emulator's specific graphics library.
type Drawer interface {
	Draw(x, y, rgb int)
}

type ppu struct {
}

func (p *ppu) step(bus memoryDevice, c uint64) error {
	return nil
}
