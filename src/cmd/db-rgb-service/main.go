package main

import (
	"github.com/docopt/docopt.go"
	log "github.com/sirupsen/logrus"
	"github.com/tarm/serial"
	"time"
	"client"
	"types"
	"os"
	"os/signal"
	"strings"
	"fmt"
	"github.com/Jeffail/gabs"
	"github.com/lucasb-eyer/go-colorful"
)

const DOC = `db-rgb-service.

Usage:
  db-rgb-service <serialPort> <configFile>
  db-rgb-service -h | --help
  db-rgb-service -v | --version

Options:
  -h --help               This message
  -v --version            Shows version
`

type PowerState bool

func (p PowerState) String() string {
	if p {
		return "ON"
	} else {
		return "OFF"
	}
}


func main() {
	opts, _ := docopt.Parse(DOC, nil, true, "0.0.1", true, true)

	serialPort := opts["<serialPort>"].(string)
	configFile := opts["<configFile>"].(string)

	deviceConfig, err := types.LoadDeviceConfig(configFile)

	if nil != err {
		panic(err)
	}

	log.Infof("Loaded config file: %s for %s @ %s", configFile, deviceConfig.ThingName, deviceConfig.Endpoint)

	portConfig := &serial.Config{Name: serialPort, Baud: 9600, ReadTimeout: time.Millisecond * 500}

	p, err := serial.OpenPort(portConfig)

	if nil != err {
		panic(err)
	}

	readBuf := make([]byte, 32)

	_, err = p.Write([]byte("AT\r\n"))

	if nil != err {
		panic(err)
	}

	_, err = p.Read(readBuf)

	if nil != err {
		panic(err)
	}

	if ! strings.Contains(string(readBuf), "OK\r\n") {
		panic(fmt.Errorf("Unxpected answer: %s", string(readBuf)))
	}

	var hue, saturation, brightness float64
	powerState := PowerState(false)

	var updateStatus func()

	deviceConfig.AddCallbackRule(&types.CallbackDefinition{
		TopicSuffix: "/shadow/update/accepted",
		JmesPath: `state.desired."Alexa.PowerController"."3".powerState`,
		Callback: func(m map[string]interface{}, val interface{}) error {
			status := val.(string)

			if "OFF" == status {
				p.Write([]byte("AT+OFF\r\n"))
			} else {
				p.Write([]byte("AT+RGB=FFFFFF\r\n"))
			}

			updateStatus()

			return nil
		},
	})

	deviceConfig.AddCallbackRule(&types.CallbackDefinition{
		TopicSuffix: "/shadow/update/accepted",
		JmesPath: `state.desired."Alexa.ColorController"."3".color`,
		Callback: func(m map[string]interface{}, val interface{}) error {
			hsb := val.(map[string]interface{})

			hue = hsb["hue"].(float64)
			saturation = hsb["saturation"].(float64)
			brightness = hsb["brightness"].(float64)

			c := colorful.Hsv(hue, saturation, brightness)

			r, g, b := c.RGB255()

			changeColorCommand := fmt.Sprintf("AT+RGB=%02X%02X%02X\r\n", byte(r), byte(g), byte(b))

			p.Write([]byte(changeColorCommand))

			updateStatus()

			return nil
		},
	})

	client, err := client.NewClient(deviceConfig)

	updateStatus = func() {
		state := gabs.New()

		state.Set(hue, "Alexa.ColorController", "3", "color", "hue")
		state.Set(saturation, "Alexa.ColorController", "3", "color", "saturation")
		state.Set(brightness, "Alexa.ColorController", "3", "color", "brightness")
		state.Set(powerState.String(), "Alexa.PowerController", "3", "powerState")

		client.ReportState(state)
	}


	if nil != err {
		panic(err)
	}

	go func() {
		client.MessagingLoop()
	}()

	quit := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		client.Close()

		log.Infof("Disconnecting mqtt")

		quit <- struct{}{}
	}()


	//<-quit

	for {
		updateStatus()

		time.Sleep(10 * time.Minute)
	}

}
