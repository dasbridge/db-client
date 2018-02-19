package main

import (
	log "github.com/sirupsen/logrus"
	"gobot.io/x/gobot/platforms/firmata"
	"github.com/docopt/docopt.go"
	"types"
	"client"
	"time"
	"gobot.io/x/gobot/drivers/i2c"
	"github.com/Jeffail/gabs"
	"os"
	"os/signal"
	"syscall"
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

	err = firmataAdaptor.Connect()

	if nil != err {
		panic(err)
	}

	bmp280 := i2c.NewBMP280Driver(firmataAdaptor, i2c.WithBus(0), i2c.WithAddress(0x76))

	bmp280.Start()

	if nil != err {
		panic(err)
	}

	quit := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-c

		log.Infof("Finishing")

		client.Stop()

		log.Infof("Disconnecting mqtt")

		quit <- struct{}{}
	}()

	client.Start()

	oldTemp := float32(0.0)

	go func() {
		for {
			temp, err := bmp280.Temperature()

			if nil != err {
				log.Warnf("Oops: %s", err)

				continue
			}

			if 0 != temp && oldTemp != temp {
				c := gabs.New()

				c.Set(temp, "Alexa.TemperatureSensor", "3", "temp")

				client.ReportState(c)
			}

			oldTemp = temp

			time.Sleep(1 * time.Minute)
		}
	}()

	<-quit

}
