package epd

import (
	"image"
	"strings"
)

// Orientation represents a screen orientation
type Orientation int

const (
	Landscape Orientation = 0
	Portrait  Orientation = 1
)

// OrientationFromString maps a string name of an
// orientation ("landscape" or "portrait" ) to its
// `Orientation`.
// It is case insensitive.
// If a match is not found it will default to landscape.
func OrientationFromString(o string) Orientation {
	if strings.ToLower(o) == "portrait" {
		return Portrait
	} else {
		return Landscape
	}
}

// Content is a DTO for transferring content to be
// shown on the display.
type Content struct {
	Title  string
	Body   string
	Footer string
	Image  image.Image
}

// Display represents the abstract high-level functions
// that can be called on an attached E-paper display
type Display interface {
	Show(content Content) (err error)
	Clear() (err error)
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
