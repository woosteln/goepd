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

// RenderContent is a id=content map that accepts
// any kind of keyed content. It is used as a map
// to populate content nodes in a template by
// matching nodes with the appropriate ID and
// setting their content.
// For the most part only string and image content
// types are supported by the default FlexRenderer
// more types may come in the future.
type RenderContent map[string]interface{}

// RenderTemplate represents a template that can
// be inflated into a layout. For the FlexRenderer
// these templates are json, specifying the layout
// stucuture of renderable nodes.
// If custom layouts or renderers are required, you
// can simply set `myCustomTemplate := RenderTemplate("{}")
// using your own template json.
type RenderTemplate string

// Render is an interface with one simple method. Render.
// Any implementers should apply the content to the provided
// template, and return an image of widthxheight dimensions
// with the content drawn to it.
// This allows either custom layouts to be used with the
// default FlexRenderer or a completely custom renderer
// to be used.
type Renderer interface {
	Render(content RenderContent, width, height int, layout RenderTemplate) (img image.Image, err error)
}

// Display represents the abstract high-level functions
// that can be called on an attached E-paper display
type Display interface {
	// Show will use the default template and renderer to update
	// the display
	Show(content RenderContent) (err error)
	// ShowWithTempalte will use specified template and default renderer
	// to update the disply. Template should be:
	// - compatible with configured renderer
	// - have id slots for speficied content
	ShowWithTemplate(content RenderContent, tpl RenderTemplate) (err error)
	// Clear clears the display.
	Clear() (err error)
	// Width returns the configured width of the display.
	Width() int
	// Height returns the configured height of the display.
	Height() int
}

// RenderOpts are used to tell the Display what renderer
// to use, and what render tempalte to use by default
type RenderOpts struct {
	Renderer Renderer
	Template RenderTemplate
}

// epd is a base struct with common properties for
// e-paper displays. Types implementing Display interface
// can use this as a base.
type epd struct {
	RendererOpts RenderOpts
	width        int
	height       int
	orientation  Orientation
	driver       Driver
}
