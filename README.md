GOEPD
=====

A golang spi driver for [Waveshare's epd displays](https://www.waveshare.com/product/displays/e-paper.htm)
using [periph.io](https://periph.io/) under the hood.

### WARNING

This is the first push so everything you read below describes intended state right now.
There may be some quirks. I'll remove the warning when its settled.

Before you dive in
------------------

This is essentially in alpha at the moment, and as such is liable to change a bit at short
notice ( although a goal is to keep the api simple ).

In addition, the only currently supported configuration is a 4.2b module ( 4.2inches in black white
and red. More coming soon...)

I'm hoping, though haven't explored it yet, that all the displays, 4.2 inches and under at least,
share a common command api. If not, support for other devices may take a bit longer.

Setting up
----------

So, we're going to assume you're connecting the display to a raspberry pi or other
maker like board. You'll need to ensure its supported in [periphs hosts](https://godoc.org/periph.io/x/periph/host)
if its not, I know there's a way to create pin mapping and host support
( I've done it before, I just can't remember how right now.)

[Waveshares hookup docs](https://www.waveshare.com/wiki/4.2inch_e-Paper_Module)
are useful, I'll describe an example setup here

An example pin connection might be as follows for a raspberry pi,
using pins collocated on the GPIO header ( [layout here](https://pinout.xyz/pinout/spi) )

(Not sure if using the HAT version of an epd display differs buch. Certainly not sure if you're
using a raw module).)

| EPD pin |        RPI Pin Name        | RPI Physical pin |                                    Description                                    |
| ------- | -------------------------- | ---------------- | --------------------------------------------------------------------------------- |
| VCC     | 3.3V                       | 17               | Using 17 as its next to default SPI pins                                          |
| GND     | GND                        | 20               | Again, closest GND to SPI pins                                                    |
| DIN     | SPI0 MOSI                  | 19               |                                                                                   |
| CLK     | SPI0 CLK                   | 23               |                                                                                   |
| CS      | SPI0 CE0                   | 24               | Default RPI chip select for SPI0                                                  |
| DC      | GPIO25 (also called BCM25) | 22               | GPIO output for Data command control. Used to tell the display when data's coming |
| RST     | GPIO24 (also called BCM24) | 18               | GPIO output for Reset control. Used to tell the board to reset                    |
| BUSY    | GPIO23 (also called BCM23) | 16               | GPIO input for busy status. Display will pull this low when its busy              |

Your choice of pins may vary, but make sure you

- use 3.3V
- connect the correct spi pins
- use addressable gpio pins for the others

Get it
------

```
go get github.com/woosteln/goepd
go get github.com/woosteln/goepd/cmd/...
```

Use it
------

The main package is a library but the `cmd/...` bit above will have installed
a utility called `goepd-show` that will let you interact with the display 
from the command line.

```bash
goepd-show --dc 25 --rst 24 --busy 23 https://loremflickr.com/400/300
```

if you want to clear the display

```
goepd-show --dc 25 --rst 24 --busy 23 clear
```

(Unless using a strange configuration, at least on rpi, spi will use "" address
to get first available bus)

If you're looking for something to use rather than building a go app, we've got
you covered.

`goepd-serve` will start an http server that will let you post content updates
to the display.

```bash
goepd-serve --dc 25 --rst 24 --busy 23 https://loremflickr.com/400/300
```

and you can then post an update to it

```bash
curl -F 'imageurl=https://loremflickr.com/400/300' $DEVICE_ADDRESS:8080/display/content
```

__!IMPORTANT!__ Don't just copy and paste, remember to substitute your device address
and the pins you've set up.

Include it
----------

Want to drive the display from your app, have a look at the src of `cmd/goepd-show/main.go`

In essence:

```
import (
  "github.com/woosteln/goepd"
)

var (
  SPI_ADDRESS = "0:0"
  DC = "25"
  RST = "24"
  BUSY = "23"
)

func main(){

  display, err := goepd.NewEPD42( goepd.Landscape, SPI_ADDRESS, DC, RST, BUSY )
  
  if err != nil {
    panic(err)
  }

  defer display.Close()

  content := goepd.Content{
    Title: "Example",
    Body: "Hello world",
    Footer: "Thank you and good night"
  }

  display.Update(content)

}
```

The renderer will try and layout content in an appropriate manner. But its pretty basic.

If you give it an image and no text, the image will be scaled to fill the display.

If you provide it text and an image, they will be laid out side by side in portrait mode
and top and bottom in landscape mode. Image will be scaled to fit its allocated area.

If you provide no image, the text will fill the space.

So, if you make a better renderer to layout text and the like, you can just send it an
already scaled image, but you might want to tell it not to dither it.

The internal driver will convert any predominantly red hues to red and anything else
will be thresholded based on its grayscale int value. >180 = white, <180 = black.

References and libraries used
-----------------------------

- [Waveshare wiki](https://www.waveshare.com/wiki/4.2inch_e-Paper_Module)
- Image manipulation using [github.com/disintegration/imaging](http://github.com/disintegration/imaging)
- Image dithering using [github.com/esimov/dithergo](https://github.com/esimov/dithergo)
- Packed using [github.com/GeertJohan/go.rice](https://github.com/GeertJohan/go.rice)
- Default font [github.com/anthonyfok/fonts-wqy-microhei](https://github.com/anthonyfok/fonts-wqy-microhei)
- [Licences](./licenses)

TODO List
---------

- Explore a better way of embedding default fonts (golangs don't work well on this display). Packages size can be quite big.
- Verify support and create constructors for other boards
- Include support for black/white only boards
- Explore supporting larger display modules
- Introduce testing
- Verify / introduce support for HATs / raw
- Improve layout / coloring in renderer

Contributing
------------

If you find this useful, shout me. If you've got a PR expanding support would love to hear about it.

I may not be the fastest to integrate changes as this is only a side-project.