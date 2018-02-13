package main

import (
	"github.com/docopt/docopt.go"
	"gobot.io/x/gobot/platforms/firmata"
	"strconv"
	"gobot.io/x/gobot/drivers/gpio"
	log "github.com/sirupsen/logrus"
	"math"
	"github.com/lucasb-eyer/go-colorful"
)

const DOC = `db-bmp280-thermometer-service.

Usage:
  db-bmp280-thermometer-service <serialPort> <h> <s> <v>
  db-bmp280-thermometer-service -h | --help
  db-bmp280-thermometer-service -v | --version

Options:
  -h --help               This message
  -v --version            Shows version
`

func main() {
	opts, _ := docopt.Parse(DOC, nil, true, "0.0.1", true, true)

	serialPort := opts["<serialPort>"].(string)

	firmataAdaptor := firmata.NewAdaptor(serialPort)

	err := firmataAdaptor.Connect()

	if nil != err {
		panic(err)
	}

	h, _ := strconv.ParseFloat(opts["<h>"].(string), 32)
	s, _ := strconv.ParseFloat(opts["<s>"].(string), 32)
	v, _ := strconv.ParseFloat(opts["<v>"].(string), 32)

	redPin := gpio.NewLedDriver(firmataAdaptor, "9")
	greenPin := gpio.NewLedDriver(firmataAdaptor, "10")
	bluePin := gpio.NewLedDriver(firmataAdaptor, "11")

	redPin.Start()
	greenPin.Start()
	bluePin.Start()

	colorful.Hsl(h, s, v)

	//c := colorful.Hsl(h, s, v)
	//
	//r, g, b := c.RGB255()

	r, g, b := hsvToRgb(h, s, v)

	log.Infof("r: %02X (%d) g: %02X (%d) b: %02X (%d)", r, r, g, g, b, b)

	redPin.Brightness(r)
	greenPin.Brightness(g)
	bluePin.Brightness(b)
}


func Round(val float64, roundOn float64, places int ) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}

func hsvToRgb(h, s, v float64) (uint8, uint8, uint8) {
	var i uint8

	var f, p, q, t float64

	//h = math.Max(0.0, math.Min(360.0, h))
	//s = math.Max(0.0, math.Min(100.0, s))
	//v = math.Max(0.0, math.Min(100.0, v))

	if (s == 0) {
		r := uint8(Round(v * 255, .5, 1))
		g := r
		b := r

		return r, g, b
	}

	h /= 60
	i = uint8(math.Floor(h))
	f = h - float64(i)
	p = v * (1 - s)
	q = v * (1 - s * f)
	t = v * (1 - s * (1 - f))

	var r, g, b uint8

	switch (i) {
	case 0:
		r = uint8(Round(255 * v, .5, 1))
		g = uint8(Round(255 * t, .5, 1))
		b = uint8(Round(255 * p, .5, 1))
	case 1:
		r = uint8(Round(255 * v, .5, 1))
		g = uint8(Round(255 * t, .5, 1))
		b = uint8(Round(255 * p, .5, 1))
	case 2:
		r = uint8(Round(255 * p, .5, 1))
		g = uint8(Round(255 * v, .5, 1))
		b = uint8(Round(255 * t, .5, 1))
	case 3:
		r = uint8(Round(255 * p, .5, 1))
		g = uint8(Round(255 * q, .5, 1))
		b = uint8(Round(255 * v, .5, 1))
	case 4:
		r = uint8(Round(255 * t, .5, 1))
		g = uint8(Round(255 * p, .5, 1))
		b = uint8(Round(255 * v, .5, 1))
	default:
		r = uint8(Round(255 * v, .5, 1))
		g = uint8(Round(255 * p, .5, 1))
		b = uint8(Round(255 * q, .5, 1))
	}

	return r, g, b
}
