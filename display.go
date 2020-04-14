package epd

import (
	"image"
)

type Orientation int

const (
	Landscape Orientation = 0
	Portrait  Orientation = 1
)

type Content struct {
	Title  string
	Body   string
	Footer string
	Image  image.Image
}

type Display interface {
	Show(content Content) (err error)
	Width() int
	Height() int
}

type epd struct {
	Renderer    Renderer
	width       int
	height      int
	orientation Orientation
	driver      Driver
}
