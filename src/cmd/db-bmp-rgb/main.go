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
	"github.com/eclipse/paho.mqtt.golang"
)

const DOC = `db-rgb.

Usage:
  db-rgb <serialPort> <configFile>
  db-rgb -h | --help
  db-rgb -v | --version

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

	var hue, saturation, brightness float32
	powerState := PowerState(false)

	deviceConfig.PublishHandler = func(i mqtt.Client, message mqtt.Message) {
		fmt.Printf("message: %s\r\n", string(message.Payload()))
	}

	client, err := client.NewClient(deviceConfig)

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

		log.Infof("Disconneting mqtt")

		quit <- struct{}{}
	}()


	//<-quit

	for {
		state := gabs.New()

		state.Set(hue, "Alexa.ColorController", "3", "color", "hue")
		state.Set(saturation, "Alexa.ColorController", "3", "color", "saturation")
		state.Set(brightness, "Alexa.ColorController", "3", "color", "brightness")
		state.Set(powerState.String(), "Alexa.PowerController", "3", "powerState")

		client.ReportState(state)

		time.Sleep(10 * time.Minute)
	}

}
