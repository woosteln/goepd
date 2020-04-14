package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/namsral/flag"
	log "github.com/sirupsen/logrus"
	epd "github.com/woosteln/goepd"
)

var (
	DC          = ""
	RESET       = ""
	BUSY        = ""
	SPI_ADDRESS = ""
	IMAGE       = ""
	LOGLEVEL    = "WARN"
)

func main() {

	fs := flag.NewFlagSetWithEnvPrefix(os.Args[0], "EPD", 0)
	fs.StringVar(&DC, "dc", DC, "Name of DC GPIO pin")
	fs.StringVar(&RESET, "reset", RESET, "Name of RESET GPIO pin")
	fs.StringVar(&BUSY, "busy", BUSY, "Name of BUSY GPIO pin")
	fs.StringVar(&SPI_ADDRESS, "spi", SPI_ADDRESS, "SPI address. Use blank for default")
	fs.StringVar(&LOGLEVEL, "loglevel", LOGLEVEL, "logging level to use")
	fs.Parse(os.Args[1:])
	IMAGE = os.Args[len(os.Args)-1]

	configureLoging(LOGLEVEL)

	imgData, err := getImageData(IMAGE)

	img, _, err := image.Decode(bytes.NewBuffer(imgData))
	if err != nil {
		fmt.Println(IMAGE, "Image data", string(imgData))
		panic(err)
	}

	content := epd.Content{
		Image: img,
	}

	display, err := epd.Epd42(epd.Landscape, SPI_ADDRESS, RESET, DC, BUSY)
	if err != nil {
		panic(err)
	}

	err = display.Show(content)
	if err != nil {
		panic(err)
	}

}

func getImageData(uri string) (data []byte, err error) {
	if strings.HasPrefix(IMAGE, "http://") || strings.HasPrefix(IMAGE, "https://") {
		response, errr := http.Get(IMAGE)
		if err != nil {
			err = errr
			return
		}
		defer response.Body.Close()
		return ioutil.ReadAll(response.Body)
	} else if _, err = os.Stat(IMAGE); err == nil {
		data, err = ioutil.ReadFile(IMAGE)
		return
	}
	err = errors.New("Could not find image, was not a url or file path")
	return
}

func configureLogging(level string) {
	switch level {
	case "INFO":
		log.SetLevel(log.InfoLevel)
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "WARN":
		log.SetLevel(log.WarnLevel)
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
	case "TRACE":
		log.SetLevel(log.TraceLevel)
	default:
		log.SetLevel(log.ErrorLevel)
	}
}
