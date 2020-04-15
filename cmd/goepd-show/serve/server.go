package serve

import (
	"image"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/labstack/echo"
	log "github.com/sirupsen/logrus"
)

type DisplayContent struct {
	Title    string      `json:"title"`
	Body     string      `json:"body"`
	Image    image.Image `json:"image"`
	ImageUrl string      `json:"imageurl"`
	Footer   string      `json:"footer"`
}

type serverData struct {
	ContentHandler
}

type EpdServer struct {
	*serverData
	Echo *echo.Echo
}

type ContentHandler func(content DisplayContent)

func (e EpdServer) HandleContent(handler ContentHandler) {
	e.ContentHandler = handler
}

func New() EpdServer {

	e := echo.New()

	server := EpdServer{serverData: &serverData{}, Echo: e}

	// Return the current display image
	e.GET("/display/content", func(c echo.Context) (err error) {
		return c.String(200, "OK")
	})

	// Update the display content
	e.POST("/display/content", func(c echo.Context) (err error) {

		content := DisplayContent{}
		if err = c.Bind(&content); err != nil {
			return
		}

		formImage, err := c.FormFile("image")

		if err != nil {
			log.Warn(err.Error())
		} else {
			file, err := formImage.Open()
			if err != nil {
				log.Warn(err.Error())
			}
			img, format, err := image.Decode(file)
			if err != nil {
				log.Warn(err.Error())
			} else {
				log.Info(format)
				content.Image = img
			}
		}

		if server.ContentHandler != nil {
			server.ContentHandler(content)
		}

		return c.String(201, "OK")
	})

	// Return the current display image
	e.GET("/", func(c echo.Context) (err error) {
		return c.String(200, "OK")
	})

	return server

}
