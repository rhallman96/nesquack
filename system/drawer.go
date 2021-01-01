package system

// Drawer is an abstraction to draw pixel data to the screen.
// It is not implemented in this package and should instead
// be implemented using the emulator's respective graphics library.
type Drawer interface {
	DrawPixel(col, row, rgb int)

	// CompleteFrame flags a single frame buffer as drawn and ready to be
	// presented in the emulator's screen.
	CompleteFrame()
}
