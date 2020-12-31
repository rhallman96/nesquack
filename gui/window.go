package gui

import (
	"github.com/rhallman96/nesquack/system"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	width  = 512
	height = 480
)

func Launch(path string) {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	window, renderer, err := sdl.CreateWindowAndRenderer(width, height, sdl.WINDOW_RESIZABLE|sdl.RENDERER_ACCELERATED)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()
	defer renderer.Destroy()

	renderer.SetLogicalSize(system.DrawWidth, system.DrawHeight)
	renderer.Clear()

	drawer := newDrawer(renderer)
	defer drawer.destroy()

	nes := system.NewNES(d)

	i := 0
	running := true
	for running {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent:
				println("Quit")
				running = false
				break
			}
		}
		for !drawer.checkComplete() {
			nes.Step()
		}
		drawer.present()
	}
}
