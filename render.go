package epd

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io/ioutil"
	"math"
	"strings"

	rice "github.com/GeertJohan/go.rice"
	"github.com/disintegration/imaging"
	dither "github.com/esimov/dithergo"
	"github.com/golang/freetype/truetype"
	"github.com/kjk/flex"
	log "github.com/sirupsen/logrus"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
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

var floydSteinbergDither = dither.Dither{
	Type: "FloydSteinberg",
	Settings: dither.Settings{
		Filter: [][]float32{
			{0.0, 0.0, 0.0, 7.0 / 48.0, 5.0 / 48.0},
			{3.0 / 48.0, 5.0 / 48.0, 7.0 / 48.0, 5.0 / 48.0, 3.0 / 48.0},
			{1.0 / 48.0, 3.0 / 48.0, 5.0 / 48.0, 3.0 / 48.0, 1.0 / 48.0},
		},
	},
}

type flexRenderEngine struct {
	font     *truetype.Font
	fontSize float64
	dpi      float64
}

func NewFlexRenderEngine(defaultFontSize float64, dpi float64, fontfile ...string) (r flexRenderEngine, err error) {

	var font *truetype.Font
	var fontBytes []byte

	if len(fontfile) > 0 {
		fontBytes, err = ioutil.ReadFile(fontfile[0])
	}

	if fontBytes == nil {
		fontbox := rice.MustFindBox("res/fonts")
		fontBytes, err = fontbox.Bytes("wqy-microhei.ttc")
	}

	font, _ = truetype.Parse(fontBytes)

	return flexRenderEngine{
		font:     font,
		fontSize: defaultFontSize,
		dpi:      dpi,
	}, nil

}

var (
	TplDefaultAuto       RenderTemplate = ""
	TplDefaulltLandscape RenderTemplate = `{
		"type": "div",
		"id": "root",
		"flexDirection":"column",
		"justifyContent":"space-between",
		"children": [
			{
				"type": "div",
				"id": "title",
				"flexDirection":"row",
				"padding":10
			},
			{
				"type": "div",
				"flexDirection": "row",
				"justifyContent": "space-evenly",
				"alignItems": "stretch",
				"alignContent": "stetch",
				"children": [
					{
						"id": "body",
						"padding":10,
						"type": "text",
						"fontsize": 1,
						"flexDirection":"column",
						"flexGrow":1
					}
				]
			},
			{
				"type": "div",
				"flexDirection":"row",
				"alignItems": "center",
				"justifyContent": "center",
				"children": [
					{
						"id": "img",
						"type": "img",
						"flexDirection":"column",
						"flexGrow": 2
					}
				]
			},
			{
				"id": "footer",
				"padding":10,
				"type": "text",
				"fontsize": 1
			}
		]
	}`
	TplDefaultPortrait RenderTemplate = `{
		"type": "div",
		"id": "root",
		"flexDirection":"column",
		"justifyContent":"space-between",
		"children": [
			{
				"type": "div",
				"id": "title",
				"flexDirection":"row"
			},
			{
				"type": "div",
				"flexDirection": "row",
				"children": [
					{
						"id": "body",
						"padding":10,
						"type": "text",
						"fontsize": 1
					},
					{
						"id": "img",
						"type": "img"
					}
				]
			},
			{
				"id": "footer",
				"padding":10,
				"type": "text",
				"fontsize": 1
			}
		]
	}`
)

func resolveTemplate(tpl RenderTemplate, width, height int) (resolved RenderTemplate) {
	if tpl == TplDefaultAuto {
		landScapeMode := width > height
		if landScapeMode {
			return TplDefaulltLandscape
		} else {
			return TplDefaultPortrait
		}
	}
	return tpl
}

func (r flexRenderEngine) Render(content RenderContent, width, height int, layout RenderTemplate) (img image.Image, err error) {

	// Determine layout template
	// If a default, work out default for aspect ratio
	// If a specific default use that
	// If a template, inflate it directly
	tpl := resolveTemplate(layout, width, height)

	// Unmarshal the template
	root := Node{}
	err = json.Unmarshal([]byte(tpl), &root)
	if err != nil {
		return
	}

	// Populate the content
	for id, item := range content {
		node, err := root.FindNodeById(id)
		if err == nil {
			node.Content = item
		}
	}

	// Layout the node structure
	config := flex.ConfigGetDefault()
	config.Context = r
	flexNode := root.Inflate(config)
	flex.CalculateLayout(flexNode, float32(width), float32(height), flex.DirectionLTR)

	// Iterate flex nodes and render them to a canvas
	colWhite := color.RGBA{255, 255, 255, 255}
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{colWhite}, image.ZP, draw.Src)

	if log.GetLevel() == log.DebugLevel {
		r.printLayout(flexNode, 0)
	}

	r.RenderNode(flexNode, image.Point{X: 0, Y: 0}, dst)

	return dst, nil
}

func (r flexRenderEngine) Measure(flexNode *flex.Node, width float32, widthMode flex.MeasureMode, height float32, heightMode flex.MeasureMode) (size flex.Size) {

	var outWidth, outHeight float64
	node := flexNode.Context.(*Node)

	switch x := node.Content.(type) {
	case string:
		requestWidth, requestHeight := r.MeasureText(x, node.FontSize, float64(width))
		outWidth = requestWidth + float64(node.Padding*2)
		outHeight = requestHeight + float64(node.Padding*2)
	case image.Image:
		requestWidth, requestHeight := r.MeasureImage(x, float64(width), widthMode, float64(height), heightMode)
		outWidth = requestWidth
		outHeight = requestHeight
	}

	if widthMode == flex.MeasureModeAtMost || widthMode == flex.MeasureModeExactly {
		outWidth = cap(outWidth, float64(width))
	}
	if heightMode == flex.MeasureModeAtMost || heightMode == flex.MeasureModeExactly {
		outHeight = cap(outHeight, float64(height))
	}

	fmt.Printf("Measure called for %s. [%.2f, %.2f][%v,%v] = [%.2f, %.2f]\n",
		node.ID,
		width,
		height,
		widthMode,
		heightMode,
		outWidth,
		outHeight,
	)

	return flex.Size{
		Width:  float32(outWidth),
		Height: float32(outHeight),
	}

}

func cap(toCap, cap float64) float64 {
	if toCap < cap {
		return toCap
	}
	return cap
}

func (r flexRenderEngine) MeasureImage(img image.Image, hintWidth float64, modeWidth flex.MeasureMode, hintHeight float64, modeHeight flex.MeasureMode) (width, height float64) {

	imgWidth, imgHeight := float64(img.Bounds().Size().X), float64(img.Bounds().Size().Y)

	if modeWidth == flex.MeasureModeUndefined && modeHeight == flex.MeasureModeUndefined {
		return imgWidth, imgHeight
	}

	if modeWidth == flex.MeasureModeExactly && modeHeight == flex.MeasureModeAtMost {
		height := scaleToTarget(hintWidth, imgWidth, imgHeight)
		if height <= hintHeight {
			return hintWidth, height
		}
		return hintWidth, hintHeight
	}

	if modeWidth == flex.MeasureModeAtMost && modeHeight == flex.MeasureModeExactly {
		width := scaleToTarget(hintHeight, imgHeight, imgWidth)
		if width <= hintWidth {
			return width, hintHeight
		}
		return hintWidth, hintHeight
	}

	return hintWidth, hintHeight

}

func scaleToTarget(target, base, applyTo float64) (applied float64) {
	scale := target / base
	return applyTo * scale
}

func (r flexRenderEngine) MeasureText(text string, fontScale float64, pixelWidth float64) (width, height float64) {

	if text == "" {
		return 0, 0
	}

	var face font.Face

	face = truetype.NewFace(r.font, &truetype.Options{
		Size:    fontScale * r.fontSize,
		DPI:     r.dpi,
		Hinting: font.HintingFull,
	})

	lineHeightPt := Int26_6ToFloat64(face.Metrics().Height)
	lineHeightPx := PointsToPixels(lineHeightPt, r.dpi)

	paras := strings.Split(text, "\n")
	var maxWidthPx float64
	var heightPx float64 = 0

	// Loop over paras. Split into lines
	for _, para := range paras {
		minIdx := 0
		maxIdx := 1
		numChars := len(para)
		var lastWidth float64
		for {
			str := para[minIdx:maxIdx]
			line := strings.TrimSpace(str)
			width26_6 := font.MeasureString(face, line)
			widthPt := Int26_6ToFloat64(width26_6)
			widthPx := PointsToPixels(widthPt, r.dpi)
			if widthPx >= pixelWidth {
				maxWidthPx = math.Max(lastWidth, maxWidthPx)
				minIdx = maxIdx - 1
				heightPx += lineHeightPx
				continue
			} else {
				if maxIdx == numChars {
					maxWidthPx = math.Max(widthPx, maxWidthPx)
					heightPx += lineHeightPx
					break
				}
				lastWidth = widthPx
				maxIdx = maxIdx + 1
			}
		}
		heightPx += lineHeightPx
	}

	return maxWidthPx, heightPx

}

func (r flexRenderEngine) RenderNode(flexNode *flex.Node, offset image.Point, dst *image.RGBA) {

	node, _ := flexNode.Context.(*Node)
	content := node.Content
	rect := image.Rectangle{
		Min: image.Point{
			X: offset.X + int(flexNode.LayoutGetLeft()) + int(flexNode.LayoutGetPadding(flex.EdgeLeft)),
			Y: offset.Y + int(flexNode.LayoutGetTop()) + int(flexNode.LayoutGetPadding(flex.EdgeTop)),
		},
		Max: image.Point{
			X: offset.X + int(flexNode.LayoutGetLeft()) + int(flexNode.LayoutGetWidth()) - int(flexNode.LayoutGetPadding(flex.EdgeRight)),
			Y: offset.Y + int(flexNode.LayoutGetTop()) + int(flexNode.LayoutGetHeight()) - int(flexNode.LayoutGetPadding(flex.EdgeBottom)),
		},
	}

	log.Debugf("Node %s: [%d,%d][%d,%d]", node.ID, rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y)

	switch x := content.(type) {
	case image.Image:
		r.drawImage(x, rect, dst)
	case string:
		r.drawText(x, node.FontSize, rect, dst)
	}

	pt := image.Point{
		X: offset.X + int(flexNode.LayoutGetLeft()),
		Y: offset.Y + int(flexNode.LayoutGetTop()),
	}
	for _, child := range flexNode.Children {
		r.RenderNode(child, pt, dst)
	}

}

func (r flexRenderEngine) printLayout(node *flex.Node, level int) {
	rNode := node.Context.(*Node)
	padding := level * 4
	log.WithFields(log.Fields{
		"level": level,
	}).Debugf("%*s%s: %.2f, %.2f, %.2f, %.2f\n",
		padding,
		"",
		rNode.ID,
		node.LayoutGetLeft(),
		node.LayoutGetTop(),
		node.LayoutGetWidth(),
		node.LayoutGetHeight())
	padding += 4
	for _, child := range node.Children {
		r.printLayout(child, padding)
	}
}

func (r flexRenderEngine) drawText(text string, scale float64, bounds image.Rectangle, dst *image.RGBA) {

	if text == "" {
		return
	}

	var face font.Face

	face = truetype.NewFace(r.font, &truetype.Options{
		Size:       scale * r.fontSize,
		DPI:        r.dpi,
		Hinting:    font.HintingNone,
		SubPixelsX: 16,
		SubPixelsY: 16,
	})

	paras := strings.Split(text, "\n")
	var maxWidthPx float64

	pixelWidth := float64(bounds.Size().X)
	colBlack := color.RGBA{0, 0, 0, 255}

	xPts := int(PixelsToPoints(float64(bounds.Min.X), r.dpi))
	yPts := int(PixelsToPoints(float64(bounds.Min.Y), r.dpi))
	dotX := fixed.I(xPts)
	dotY := fixed.I(yPts) + face.Metrics().Ascent
	draw := &font.Drawer{
		Dst:  dst,
		Src:  &image.Uniform{colBlack},
		Face: face,
		Dot: fixed.Point26_6{
			X: dotX,
			Y: dotY,
		},
	}

	// Loop over paras. Split into lines
	for _, para := range paras {
		minIdx := 0
		maxIdx := 1
		numChars := len(para)
		var lastWidth float64
		var lastLine string
		for {
			str := para[minIdx:maxIdx]
			line := strings.TrimSpace(str)
			width26_6 := font.MeasureString(face, line)
			widthPt := Int26_6ToFloat64(width26_6)
			widthPx := PointsToPixels(widthPt, r.dpi)
			if widthPx > pixelWidth {
				maxWidthPx = math.Max(lastWidth, maxWidthPx)
				minIdx = maxIdx - 1
				draw.DrawBytes([]byte(lastLine))
				draw.Dot.Y = draw.Dot.Y + face.Metrics().Height
				draw.Dot.X = dotX
				continue
			} else {
				if maxIdx == numChars {
					maxWidthPx = math.Max(lastWidth, maxWidthPx)
					draw.DrawBytes([]byte(line))
					draw.Dot.Y = draw.Dot.Y + face.Metrics().Height
					draw.Dot.X = dotX
					break
				}
				lastWidth = widthPx
				lastLine = line
				maxIdx = maxIdx + 1
			}
		}
		draw.Dot.Y = draw.Dot.Y + face.Metrics().Height
		draw.Dot.X = dotX
	}

}

func (r flexRenderEngine) drawImage(img image.Image, bounds image.Rectangle, dst *image.RGBA) {

	targetWidth, targetHeight := float64(bounds.Size().X), float64(bounds.Size().Y)
	width := float64(img.Bounds().Size().X)
	height := float64(img.Bounds().Size().Y)
	scaledHeight := scaleToTarget(targetWidth, width, height)
	scaledWidth := scaleToTarget(targetHeight, height, width)
	var resizeWidth, resizeHeight int
	if scaledHeight <= targetHeight {
		resizeWidth = int(targetWidth)
		resizeHeight = int(scaledHeight)
	} else if scaledWidth <= targetWidth {
		resizeWidth = int(scaledWidth)
		resizeHeight = int(targetHeight)
	} else {
		resizeWidth = int(targetWidth)
		resizeHeight = int(targetHeight)
	}

	var xOffset, yOffset int
	xOffset = int((targetWidth / 2)) - (resizeWidth / 2)
	yOffset = int((targetHeight / 2)) - (resizeHeight / 2)
	newBounds := image.Rectangle{
		image.Point{
			X: bounds.Min.X + xOffset,
			Y: bounds.Min.Y + yOffset,
		},
		image.Point{
			X: bounds.Min.X + xOffset + resizeWidth,
			Y: bounds.Min.Y + yOffset + resizeHeight,
		},
	}

	log.Debugf("Scaling image to [%.0f, %.0f]", targetWidth, targetHeight)

	updateImg := imaging.Resize(img, resizeWidth, resizeHeight, imaging.Lanczos)
	multiplier := 0.3 // Smaller for smaller images
	// outputMono := floydSteinbergDither.Monochrome(updateImg, float32(multiplier))
	outputMono := atkinsonDither.Monochrome(updateImg, float32(multiplier))
	draw.Draw(dst, newBounds, outputMono, image.ZP, draw.Src)
}

func PixelsToPoints(pixels, dpi float64) float64 {
	inches := pixels / dpi
	return 72 * inches
}

func PointsToPixels(points, dpi float64) float64 {
	inches := points / 72
	return dpi * inches
}

func Int26_6ToFloat64(x fixed.Int26_6) float64 {
	const shift, mask = 6, 1<<6 - 1
	if x >= 0 {
		return float64(x>>shift) + float64(x&mask)/72
	}
	x = -x
	if x >= 0 {
		return float64(x>>shift) + float64(x&mask)/72
	}
	return math.MaxFloat64
}
