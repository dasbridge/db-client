package main

import (
	log "github.com/sirupsen/logrus"
	"gobot.io/x/gobot/platforms/firmata"
	"github.com/docopt/docopt.go"
	"types"
	"client"
	"gobot.io/x/gobot/drivers/gpio"
	"gobot.io/x/gobot"
	"time"
	"gobot.io/x/gobot/drivers/i2c"
	"github.com/Jeffail/gabs"
)

const DOC = `db-bmp280-thermometer-service.

Usage:
  db-bmp280-thermometer-service [options] <configFile>
  db-bmp280-thermometer-service -h | --help
  db-bmp280-thermometer-service -v | --version

Options:
  -s --serial SERIALPORT  Serial Port [default: /dev/ttyACM0]
  -h --help               This message
  -v --version            Shows version
`

func main() {
	opts, _ := docopt.Parse(DOC, nil, true, "0.0.1", true, true)

	configFile := opts["<configFile>"].(string)

	deviceConfig, err := types.LoadDeviceConfig(configFile)

	if nil != err {
		panic(err)
	}

	log.Infof("Loaded config file: %s for %s @ %s", configFile, deviceConfig.ThingName, deviceConfig.Endpoint)

	client, err := client.NewClient(deviceConfig)

	serialPort := opts["--serial"].(string)

	firmataAdaptor := firmata.NewAdaptor(serialPort)

	led := gpio.NewLedDriver(firmataAdaptor, "13")

	bmp280 := i2c.NewBMP280Driver(firmataAdaptor, i2c.WithBus(0), i2c.WithAddress(0x76))

	var temp float32

	work := func() {
		gobot.Every(5 * time.Second, func() {
			led.Toggle()

			var err error

			temp, err = bmp280.Temperature()

			if nil != err {
				log.Warnf("WARN: %s", err)
			}
		})
	}

	robot := gobot.NewRobot("bot",
		[]gobot.Connection{firmataAdaptor},
		[]gobot.Device{led,bmp280},
		work,
	)

	go func() {
		robot.Start()
	}()

	defer robot.Stop()

	//client.Debug = true

	if nil != err {
		panic(err)
	}

	go func() {
		defer client.Close()

		client.MessagingLoop()
	}()

	for {
		if 0 != temp {
			c := gabs.New()

			c.Set(temp,"Alexa.TemperatureSensor", "3", "temp")

			client.ReportState(c)
		}

		time.Sleep(5 * time.Second)
	}
}
