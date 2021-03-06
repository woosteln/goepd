package epd

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"github.com/disintegration/imaging"
	log "github.com/sirupsen/logrus"
	"periph.io/x/periph/conn/gpio"
)

type Command []byte

var (
	PANEL_SETTING                  Command = []byte{0x00}
	POWER_SETTING                  Command = []byte{0x01}
	POWER_OFF                      Command = []byte{0x02}
	POWER_OFF_SEQUENCE_SETTING     Command = []byte{0x03}
	POWER_ON                       Command = []byte{0x04}
	POWER_ON_MEASURE               Command = []byte{0x05}
	BOOSTER_SOFT_START             Command = []byte{0x06}
	DEEP_SLEEP                     Command = []byte{0x07}
	DATA_START_TRANSMISSION_1      Command = []byte{0x10}
	DATA_STOP                      Command = []byte{0x11}
	DISPLAY_REFRESH                Command = []byte{0x12}
	DATA_START_TRANSMISSION_2      Command = []byte{0x13}
	VCOM_LUT                       Command = []byte{0x20}
	W2W_LUT                        Command = []byte{0x21}
	B2W_LUT                        Command = []byte{0x22}
	W2B_LUT                        Command = []byte{0x23}
	B2B_LUT                        Command = []byte{0x24}
	PLL_CONTROL                    Command = []byte{0x30}
	TEMPERATURE_SENSOR_CALIBRATION Command = []byte{0x40}
	TEMPERATURE_SENSOR_SELECTION   Command = []byte{0x41}
	TEMPERATURE_SENSOR_WRITE       Command = []byte{0x42}
	TEMPERATURE_SENSOR_READ        Command = []byte{0x43}
	VCOM_AND_DATA_INTERVAL_SETTING Command = []byte{0x50}
	LOW_POWER_DETECTION            Command = []byte{0x51}
	TCON_SETTING                   Command = []byte{0x60}
	RESOLUTION_SETTING             Command = []byte{0x61}
	GSST_SETTING                   Command = []byte{0x65}
	GET_STATUS                     Command = []byte{0x71}
	AUTO_MEASURE_VCOM              Command = []byte{0x80}
	VCOM_VALUE                     Command = []byte{0x81}
	VCM_DC_SETTING                 Command = []byte{0x82}
	PARTIAL_WINDOW                 Command = []byte{0x90}
	PARTIAL_IN                     Command = []byte{0x91}
	PARTIAL_OUT                    Command = []byte{0x92}
	PROGRAM_MODE                   Command = []byte{0xA0}
	ACTIVE_PROGRAM                 Command = []byte{0xA1}
	READ_OTP_DATA                  Command = []byte{0xA2}
	POWER_SAVING                   Command = []byte{0xE3}
)

type smallEpd struct {
	epd
	RESET      string
	DC         string
	BUSY       string
	SPIAddress string
}

// Epd42 returns a display suitable for driving a waveshare
// 4.2" epaper screen.
func Epd42(orientation Orientation, spiAddress, reset, dc, busy string, renderOpts ...RenderOpts) (display Display, err error) {

	width := 400
	height := 300

	var renderOpt RenderOpts
	if len(renderOpts) == 0 {
		// We know that using the default font
		// will work, so no need to handle error
		renderer, _ := NewFlexRenderEngine(11, 72)
		renderOpt = RenderOpts{
			Renderer: renderer,
			Template: TplDefaultAuto,
		}
	} else {
		renderOpt = renderOpts[0]
	}

	base := epd{
		RendererOpts: renderOpt,
		width:        width,
		height:       height,
		orientation:  orientation,
		driver:       SpiGpioDriver(),
	}

	sepd := smallEpd{
		epd:        base,
		RESET:      reset,
		DC:         dc,
		BUSY:       busy,
		SPIAddress: spiAddress,
	}

	err = sepd.init()

	return sepd, err

}

func (display smallEpd) init() (err error) {

	err = display.driver.Init(display.SPIAddress, display.RESET, display.DC, display.BUSY)
	if err != nil {
		err = fmt.Errorf("Error initialising driver. %s", err)
		return
	}

	if err = display.driver.Pin(display.RESET).Out(gpio.Low); err != nil {
		err = fmt.Errorf("Error setting up RESET pin: %s", err.Error())
	} else if err = display.driver.Pin(display.DC).Out(gpio.Low); err != nil {
		err = fmt.Errorf("Error setting up DC pin: %s", err.Error())
	} else if err := display.driver.Pin(display.BUSY).In(gpio.PullDown, gpio.NoEdge); err != nil {
		err = fmt.Errorf("Error setting up BUSY pin: %s", err.Error())
	}

	if err != nil {
		err = fmt.Errorf("Could not set up display: %s", err.Error())
		display.epd.driver.Close()
	}

	return

}

func (display smallEpd) Height() int {
	return display.height
}

func (display smallEpd) Width() int {
	return display.width
}

func (display smallEpd) Show(content RenderContent) (err error) {
	return display.ShowWithTemplate(content, display.RendererOpts.Template)
}

func (display smallEpd) ShowWithTemplate(content RenderContent, tpl RenderTemplate) (err error) {

	var width, height int

	if display.orientation == Landscape {
		width = display.width
		height = display.height
	} else {
		width = display.height
		height = display.width
	}

	opts := display.RendererOpts
	img, err := opts.Renderer.Render(content, width, height, tpl)

	if err = display.prepare(); err != nil {
		return
	}

	if err = display.show(img); err != nil {
		return
	}

	if err = display.sleep(); err != nil {
		return
	}

	return
}

func (display smallEpd) prepare() (err error) {
	log.Debug("EPD42 Prepare")

	if err = display.reset(); err != nil {
		return
	}

	if err = display.sendCommand(BOOSTER_SOFT_START); err != nil {
		return
	}

	if err = display.sendData([]byte{0x17}); err != nil {
		return
	}

	if err = display.sendData([]byte{0x17}); err != nil {
		return
	}

	if err = display.sendData([]byte{0x17}); err != nil {
		return
	} // 07 0f 17 1f 27 2F 37 2f

	if err = display.sendCommand(POWER_ON); err != nil {
		return
	}

	if err = display.waitUntilIdle(); err != nil {
		return
	}

	if err = display.sendCommand(PANEL_SETTING); err != nil {
		return
	}

	if err = display.sendData([]byte{0x0F}); err != nil {
		return
	} // LUT from OTP

	log.Debug("EPD42 Prepare End")
	return
}

// Display pushes the provided buffers to display
// imageBlack is the black pixel buffer, imageRed is the red pixel buffer
func (display smallEpd) show(img image.Image) (err error) {
	log.Debug("EPD42 Show")
	displayHorizontal := display.Width() >= display.Height()
	imageHorizontal := img.Bounds().Dx() >= img.Bounds().Dy()
	if displayHorizontal != imageHorizontal {
		// Rotate image 90
		img = imaging.Rotate90(img)
	}
	if img.Bounds().Dx() != display.Width() || img.Bounds().Dy() != display.Height() {
		img = imaging.Resize(img, display.Width(), display.Height(), imaging.Lanczos)
	}
	imageblack, imagered := display.convertImage(img)

	if err = display.sendCommand(DATA_START_TRANSMISSION_1); err != nil {
		return
	}

	if err = display.sendData(imageblack); err != nil {
		return
	}

	if err = display.sendCommand(DATA_START_TRANSMISSION_2); err != nil {
		return
	}

	if err = display.sendData(imagered); err != nil {
		return
	}

	if err = display.sendCommand(DISPLAY_REFRESH); err != nil {
		return
	}

	if err = display.waitUntilIdle(); err != nil {
		return
	}

	log.Debug("EPD42 Show End")
	return
}

// Clear clears the display
func (display smallEpd) Clear() (err error) {
	log.Debug("EPD42 Clear")

	if err = display.sendCommand(DATA_START_TRANSMISSION_1); err != nil {
		return
	}

	// TODO: Verify that this is enough bits
	for i := 0; i < display.Width()*display.Height()/8; i++ {
		if err = display.sendData([]byte{0xFF}); err != nil {
			return
		}
	}

	if err = display.sendCommand(DATA_START_TRANSMISSION_2); err != nil {
		return
	}

	for i := 0; i < display.Width()*display.Height()/8; i++ {
		if err = display.sendData([]byte{0xFF}); err != nil {
			return
		}
	}

	if err = display.sendCommand(DISPLAY_REFRESH); err != nil {
		return
	}

	if err = display.waitUntilIdle(); err != nil {
		return
	}

	log.Debug("EPD42 Clear End")
	return
}

// Sleep sends the display to sleep
func (display smallEpd) sleep() (err error) {
	log.Debug("EPD42  Sleep")

	if err = display.sendCommand(VCOM_AND_DATA_INTERVAL_SETTING); err != nil {
		return
	}

	if err = display.sendData([]byte{0xF7}); err != nil {
		return
	} // border floating

	if err = display.sendCommand(POWER_OFF); err != nil {
		return
	}

	if err = display.waitUntilIdle(); err != nil {
		return
	}

	if err = display.sendCommand(DEEP_SLEEP); err != nil {
		return
	}

	if err = display.sendData([]byte{0xA5}); err != nil {
		return
	} // check code

	log.Debug("EPD42 Sleep End")
	return
}

// Reset resets registers?
func (display smallEpd) reset() (err error) {
	log.Debug("EPD42 Reset")

	if err = display.driver.DigitalWrite(display.RESET, gpio.High); err != nil {
		return
	}
	time.Sleep(200 * time.Millisecond)

	if err = display.driver.DigitalWrite(display.RESET, gpio.Low); err != nil {
		return
	}
	time.Sleep(200 * time.Millisecond)

	if err = display.driver.DigitalWrite(display.RESET, gpio.High); err != nil {
		return
	}
	time.Sleep(200 * time.Millisecond)

	log.Debug("EPD42 Reset End")
	return
}

// SendCommand sends a command to the device ( a specific byte )
// command must be a valid EPD command
func (display smallEpd) sendCommand(command Command) (err error) {
	log.Debug("EPD42 SendCommand")

	if err = display.driver.DigitalWrite(display.driver.CS(), gpio.Low); err != nil {
		return
	}

	if err = display.driver.DigitalWrite(display.DC, gpio.Low); err != nil {
		return
	}

	if err = display.driver.Write(command); err != nil {
		return
	}

	if err = display.driver.DigitalWrite(display.driver.CS(), gpio.High); err != nil {
		return
	}

	log.Debug("EPD42 SendCommand End")
	return
}

// SendData writes data to the SPI connection of the device
func (display smallEpd) sendData(data []byte) (err error) {
	log.Debug("EPD42 SendData")
	if err = display.driver.DigitalWrite(display.driver.CS(), gpio.Low); err != nil {
		return
	}

	if err = display.driver.DigitalWrite(display.DC, gpio.High); err != nil {
		return
	}

	if err = display.driver.Write(data); err != nil {
		return
	}

	if err = display.driver.DigitalWrite(display.driver.CS(), gpio.High); err != nil {
		return
	}

	log.Debug("EPD42 SendData End")
	return
}

// WaitUntilIdle blocks until the device becomes available
func (display smallEpd) waitUntilIdle() (err error) {
	log.Debug("EPD42 WaitUntilIdle")
	for {
		busy, err := display.driver.DigitalRead(display.BUSY)
		if !busy {
			break
		}
		if err != nil {
			fmt.Printf("Error checking bust %s\n", err.Error())
		}
		fmt.Printf(".")
		time.Sleep(200 * time.Millisecond)
	}
	log.Debug("EPD42 WaitUntilIdle End")
	return
}

// GetBuffer converts the given image into a buffer
// fitted to the size of the e-paper display
func (display smallEpd) convertImage(image image.Image) (black []byte, red []byte) {
	// Resize image to width and height
	// Each pixel in image is turned into a bit
	// which says 1 or 0
	// Create two buffers of (w*h)/8 bytes
	// TODO: Allow for other colors. Switch to HSL mode and
	// calculate by hue
	w := display.Width()
	h := display.Height()
	s := (w * h) / 8
	blackBuf := make([]byte, s)
	redBuf := make([]byte, s)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			pixelIdx := ((y * w) + x)
			byteIdx := pixelIdx / 8
			bitIdx := uint(7 - pixelIdx%8)
			pix := image.At(x, y)
			rgba := color.RGBAModel.Convert(pix).(color.RGBA)
			gray := color.GrayModel.Convert(pix).(color.Gray)
			// Flip all bits and mask with 0xFF. Divide by 0xFF to get 1 as last bit for black, 0 for anything else. Then XOR it.
			// black := (((^rgba.R ^ rgba.B ^ rgba.G) & 0xFF) / 0xFF) ^ 0x01 // Black is 1 (white) if not absolute black
			// red := ((rgba.R &^ rgba.B &^ rgba.G) / 0xFF) ^ 0x01           // Red is 1 if only full saturation red. Otherwise 0
			black := byte(0x00)
			if gray.Y > 180 {
				black = 0x01
			}
			red := byte(0x01)
			if rgba.B < 180 && rgba.G < 180 && rgba.R > 180 {
				red = 0x00
			}
			blackBuf[byteIdx] |= black << bitIdx
			redBuf[byteIdx] |= red << bitIdx
		}
	}
	// Dither and do another loop for black?
	return blackBuf, redBuf
}
