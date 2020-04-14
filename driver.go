package epd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/conn/physic"
	"periph.io/x/periph/conn/spi"
	"periph.io/x/periph/conn/spi/spireg"
	"periph.io/x/periph/host"
)

type Driver interface {
	Init(spiAddress string, pins ...string) (err error)
	DigitalRead(pin string) (bool, error) // Read value from a pin
	DigitalWrite(pin string, level gpio.Level) error
	Pin(pin string) gpio.PinIO
	CS() string
	Write(data []byte) error
	Close() (err error)
}

type gpioSpiData struct {
	Pins map[string]gpio.PinIO
	P    spi.PortCloser
	C    spi.Conn
	cs   string
}

// gpioSpiDriver is a generic driver that allows
// a higher level driver to set pins and interact
// with an spi device, covering digital read and
// write
type gpioSpiInterface struct {
	*gpioSpiData
}

func SpiGpioDriver() Driver {
	return gpioSpiInterface{
		&gpioSpiData{},
	}
}

func (g gpioSpiInterface) Init(spiAddress string, pins ...string) (err error) {

	if _, err = host.Init(); err != nil {
		return
	}

	pinMap := make(map[string]gpio.PinIO)

	for _, pinName := range pins {
		pinMap[pinName] = gpioreg.ByName(pinName)
	}

	for key, pin := range pinMap {
		if pin == nil {
			err = fmt.Errorf("Could not find PIN %s", key)
			return
		}
	}

	p, err := spireg.Open(spiAddress)
	if err != nil {
		err = fmt.Errorf("Driver error opening SPI %s: %s\n", spiAddress, err.Error())
		return
	}

	// TODO: This might not be applicable to all board configs
	c, err := p.Connect(2*physic.MegaHertz, spi.Mode0, 8)
	if err != nil {
		err = fmt.Errorf("Driver error connecting SPI %s\n", err.Error())
		p.Close()
		return
	}

	if c, ok := c.(spi.Pins); ok {
		log.Debugf("  SPI CLK : %s", c.CLK())
		log.Debugf("  SPI MOSI: %s", c.MOSI())
		log.Debugf("  SPI MISO: %s", c.MISO())
		log.Debugf("  SPI CS  : %s", c.CS())
		g.cs = c.CS().Name()
	} else {
		err = fmt.Errorf("Driver error verifying SPI pins\n")
		p.Close()
		return
	}

	g.Pins = pinMap
	g.P = p
	g.C = c

	return

}

func (g gpioSpiInterface) CS() string {
	return g.cs
}

func (g gpioSpiInterface) DigitalRead(pin string) (bool, error) {
	if p, ok := g.Pins[pin]; ok {
		return p.Read() == gpio.High, nil
	}
	return false, fmt.Errorf("Could not read. Pin %s does not exist", pin)
}

func (g gpioSpiInterface) DigitalWrite(pin string, level gpio.Level) error {
	if pin == g.cs {
		pins := g.C.(spi.Pins)
		if err := pins.CS().Out(level); err != nil {
			return err
		}
	}
	if p, ok := g.Pins[pin]; ok {
		return p.Out(level)
	}
	return fmt.Errorf("Could not write. Pin %s does not exist. SPI cs pin is %s", pin, g.cs)
}

func (g gpioSpiInterface) Write(data []byte) error {
	read := make([]byte, len(data))
	// for _, bt := range data {
	if err := g.C.Tx(data, read); err != nil {
		return err
	}
	// }
	return nil
}

func (g gpioSpiInterface) Pin(pin string) gpio.PinIO {
	return g.Pins[pin]
}

func (g gpioSpiInterface) Close() (err error) {
	return g.P.Close()
}
