package gui

import (
	"github.com/rhallman96/nesquack/system"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	width  = 512
	height = 480
)

func Launch(rom []uint8) {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	window, err := sdl.CreateWindow("nesquack", 0, 0, width, height, 0)
	if err != nil {
		panic(err)
	}
	renderer, err := sdl.CreateRenderer(window, 0, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)
	if err != nil {
		panic(err)
	}

	defer window.Destroy()
	defer renderer.Destroy()

	renderer.SetLogicalSize(system.DrawWidth, system.DrawHeight)
	renderer.Clear()

	j1 := &controller{}

	drawer := newDrawer(renderer)
	defer drawer.destroy()

	nes, err := system.NewNES(rom, drawer, j1)
	if err != nil {
		panic(err)
	}

	running := true
	for running {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent:
				running = false
				break
			}
		}
		for !drawer.checkComplete() {
			err := nes.Step()
			if err != nil {
				panic(err)
			}
		}
		drawer.present()
	}
}
