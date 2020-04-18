package main

import (
	"fmt"
	"image"
	"net/http"
	"os"

	"github.com/namsral/flag"
	log "github.com/sirupsen/logrus"
	epd "github.com/woosteln/goepd"
	"github.com/woosteln/goepd/cmd/epd-serve/serve"
)

var (
	ADDR        = "0.0.0.0"
	PORT        = 8080
	DC          = ""
	RESET       = ""
	BUSY        = ""
	SPI_ADDRESS = ""
	ORIENTATION = ""
	LOGLEVEL    = "WARN"
)

func main() {

	fs := flag.NewFlagSetWithEnvPrefix(os.Args[0], "EPD", 0)
	fs.StringVar(&ADDR, "addr", ADDR, "Host address to bind to")
	fs.IntVar(&PORT, "port", PORT, "Host port to bind to")
	fs.StringVar(&DC, "dc", DC, "DC GPIO pin")
	fs.StringVar(&RESET, "rst", RESET, "RST GPIO pin")
	fs.StringVar(&BUSY, "busy", BUSY, "BUSY GPIO pin")
	fs.StringVar(&SPI_ADDRESS, "spi", SPI_ADDRESS, "Spi bus address. Omit or leave blank for default (recommended)")
	fs.StringVar(&ORIENTATION, "orientation", ORIENTATION, "Orientation of attached display. 'portrait' or 'landscape'")
	fs.StringVar(&LOGLEVEL, "loglevel", LOGLEVEL, "Log level for app")
	fs.Parse(os.Args[1:])

	display, err := epd.Epd42(epd.OrientationFromString(ORIENTATION), SPI_ADDRESS, RESET, DC, BUSY)
	if err != nil {
		panic(err)
	}

	server := serve.New()

	server.HandleContent(func(content serve.DisplayContent) {

		log.Debugf("Got content from server %v", content)

		if content.Image == nil && content.ImageUrl != "" {
			content.Image, err = loadImageFromUrl(content.ImageUrl)
		}

		display.Show(epd.RenderContent{
			"title":  content.Title,
			"body":   content.Body,
			"image":  content.Image,
			"footer": content.Footer,
		})
	})

	server.Echo.Server.Addr = fmt.Sprintf("%s:%d", ADDR, PORT)

	err = server.Echo.Server.ListenAndServe()
	if err != nil {
		panic(err)
	}

}

func loadImageFromUrl(url string) (img image.Image, err error) {
	response, errr := http.Get(url)
	if err != nil {
		err = errr
		return
	}
	defer response.Body.Close()
	img, _, err = image.Decode(response.Body)
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
