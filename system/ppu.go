package system

// Drawer is an abstraction to draw pixel data to the screen.
// It is not implemented in this package and should instead
// be implemented with the emulator's specific graphics library.
type Drawer interface {
	Draw(x, y, rgb int)
}

type ppu struct {
}
