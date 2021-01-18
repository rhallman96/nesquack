package gui

import (
	"github.com/veandco/go-sdl2/sdl"
)

type controller struct {
}

func (c *controller) Up() bool {
	return readKey(sdl.SCANCODE_UP)
}

func (c *controller) Down() bool {
	return readKey(sdl.SCANCODE_DOWN)
}

func (c *controller) Left() bool {
	return readKey(sdl.SCANCODE_LEFT)
}

func (c *controller) Right() bool {
	return readKey(sdl.SCANCODE_RIGHT)
}

func (c *controller) A() bool {
	return readKey(sdl.SCANCODE_X)
}

func (c *controller) B() bool {
	return readKey(sdl.SCANCODE_Z)
}

func (c *controller) Start() bool {
	return readKey(sdl.SCANCODE_RETURN)
}

func (c *controller) Select() bool {
	return readKey(sdl.SCANCODE_RSHIFT)
}

func readKey(key int) bool {
	keyboard := sdl.GetKeyboardState()
	if keyboard[key] != 0 {
		return true
	}
	return false
}
