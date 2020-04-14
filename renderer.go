package epd

import (
	"image"
	"image/color"
	"image/draw"
	"io/ioutil"

	rice "github.com/GeertJohan/go.rice"
	"github.com/disintegration/imaging"
	dither "github.com/esimov/dithergo"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

var (
	utf8FontFile = "wqy-microhei.ttc"
	utf8FontSize = float64(12.0)
	spacing      = float64(1.5)
	dpi          = float64(120)
	ctx          = new(freetype.Context)
	utf8Font     = new(truetype.Font)
)

var atkinsonDither = dither.Dither{
	Type: "Atkinson",
	Settings: dither.Settings{
		Filter: [][]float32{
			{0.0, 0.0, 1.0 / 8.0, 1.0 / 8.0},
			{1.0 / 8.0, 1.0 / 8.0, 1.0 / 8.0, 0.0},
			{0.0, 1.0 / 8.0, 0.0, 0.0},
		},
	},
}

var sierraLiteDither = dither.Dither{
	Type: "Sierra-Lite",
	Settings: dither.Settings{
		Filter: [][]float32{
			{0.0, 0.0, 2.0 / 4.0},
			{1.0 / 4.0, 1.0 / 4.0, 0.0},
			{0.0, 0.0, 0.0},
		},
	},
}

type Renderer struct {
	Font *truetype.Font
}

func LoadRenderer(fontFile ...string) (renderer Renderer, err error) {

	var fontBytes []byte

	if len(fontFile) == 0 {
		fontbox := rice.MustFindBox("res/fonts")
		fontBytes, err = fontbox.Bytes("wqy-microhei.ttc")
	} else {
		fontBytes, err = ioutil.ReadFile(fontFile[0])
	}
	if err != nil {
		return
	}

	utf8Font, err = freetype.ParseFont(fontBytes)
	if err != nil {
		return
	}

	renderer = Renderer{
		Font: utf8Font,
	}

	return

}

func (r Renderer) Render(content Content, width, height int) (img image.Image, err error) {

	// colRed := color.RGBA{255, 0, 0, 255}
	colWhite := color.RGBA{255, 255, 255, 255}
	colBlack := color.RGBA{0, 0, 0, 255}

	// Prepare blank canvas
	// TODO: Switch based on orientation
	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	draw.Draw(dst, dst.Bounds(), &image.Uniform{colWhite}, image.ZP, draw.Src)

	if content.Title != "" {
		drawText(dst, 10, 20, colBlack, content.Title)
	}

	if content.Body != "" {
		drawText(dst, 10, 40, colBlack, content.Body)
	}

	if content.Footer != "" {
		drawText(dst, 10, height-20, colBlack, content.Footer)
	}

	imageOnly := content.Title == "" && content.Body == "" && content.Footer == ""

	if content.Image != nil {
		var maxWidth, x int
		if imageOnly {
			maxWidth = width - 20
			x = 10
		} else {
			maxWidth = (width - 40) / 2
			x = width/2 + 10
		}
		updateImg := imaging.Resize(content.Image, maxWidth, 0, imaging.Lanczos)
		multiplier := 0.5 // Smaller for smaller images
		outputMono := atkinsonDither.Monochrome(updateImg, float32(multiplier))
		pt := image.Point{x, 10}
		dstRct := image.Rectangle{pt, pt.Add(outputMono.Bounds().Size())}
		draw.Draw(dst, dstRct, outputMono, image.ZP, draw.Src)
	}

	return dst, nil

}

func drawLabel(img *image.RGBA, x, y int, col color.Color, label string) {
	point := fixed.Point26_6{X: fixed.Int26_6(x * 64), Y: fixed.Int26_6(y * 64)}
	fontWriter := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	fontWriter.DrawString(label)
}

func drawText(img *image.RGBA, x, y int, col color.Color, label string) {
	// Work out multilines with https://stackoverflow.com/questions/29105540/aligning-text-in-golang-with-truetype
	ctx = freetype.NewContext()
	ctx.SetDPI(dpi) //screen resolution in Dots Per Inch
	ctx.SetFont(utf8Font)
	ctx.SetHinting(font.HintingFull)
	ctx.SetFontSize(utf8FontSize) //font size in points
	ctx.SetClip(img.Bounds())
	ctx.SetDst(img)
	ctx.SetSrc(image.NewUniform(col))
	pt := freetype.Pt(x, y+int(ctx.PointToFixed(utf8FontSize)>>6))
	_, _ = ctx.DrawString(label, pt)
	// pt.Y += ctx.PointToFixed(utf8FontSize * spacing)
}
