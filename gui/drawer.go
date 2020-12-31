package gui

import (
	"github.com/rhallman96/nesquack/system"
	"github.com/veandco/go-sdl2/sdl"
)

type drawer struct {
	renderer *sdl.Renderer
	texture  *sdl.Texture
	pixels   []byte
	complete bool
}

func newDrawer(renderer *sdl.Renderer) *drawer {
	texture, err := renderer.CreateTexture(sdl.PIXELFORMAT_RGB24, sdl.TEXTUREACCESS_STREAMING,
		system.DrawWidth, system.DrawHeight)
	if err != nil {
		panic(err)
	}
	pixels := make([]byte, system.DrawWidth*system.DrawHeight*3)

	return &drawer{
		renderer: renderer,
		texture:  texture,
		pixels:   pixels,
	}
}

func (d *drawer) DrawPixel(col, row, rgb int) {
	i := ((row * system.DrawWidth) + col) * 3
	d.pixels[i] = byte((rgb >> 16) & 0xff)  //red
	d.pixels[i+1] = byte((rgb >> 8) & 0xff) //green
	d.pixels[i+2] = byte(rgb & 0xff)        // blue
}

func (d *drawer) CompleteFrame() {
	d.complete = true
}

func (d *drawer) checkComplete() bool {
	if d.complete {
		d.complete = false
		return true
	}
	return false
}

func (d *drawer) present() {
	d.texture.Update(nil, d.pixels, system.DrawWidth*3)
	d.renderer.Copy(d.texture, nil, nil)
	d.renderer.Present()
}

func (d *drawer) destroy() {
	d.texture.Destroy()
}
