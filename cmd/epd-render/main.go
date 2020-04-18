package main

import (
	"image"
	"image/png"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/namsral/flag"
	epd "github.com/woosteln/goepd"
)

var (
	LOGLEVEL = "INFO"
)

func main() {

	fs := flag.NewFlagSetWithEnvPrefix(os.Args[0], "EPD", 0)
	fs.StringVar(&LOGLEVEL, "loglevel", LOGLEVEL, "loglevel for app.")
	fs.Parse(os.Args[1:])

	configureLogging(LOGLEVEL)

	engine, err := epd.NewFlexRenderEngine(9, 72)
	if err != nil {
		panic(err)
	}

	showImg, _ := loadImageFromUrl("https://loremflickr.com/400/300")

	content := epd.RenderContent{
		"title": "Hello World!",
		"body": `Here's a lovely picture that's probably of a cat.
It's really hard to tell with this flickr account because it always sends you something random.`,
		"footer": "Hope you like it",
		"img":    showImg,
	}

	img, err := engine.Render(content, 400, 300, epd.TplDefaultAuto)
	if err != nil {
		log.Fatal("Render error", err)
	}

	f, err := os.Create("image.png")
	if err != nil {
		log.Fatal(err)
	}

	if err := png.Encode(f, img); err != nil {
		f.Close()
		log.Fatal(err)
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
