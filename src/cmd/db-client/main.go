package main

import (
	"github.com/docopt/docopt.go"
	"types"
	log "github.com/sirupsen/logrus"
	"client"
	"bufio"
	"os"
	"github.com/Jeffail/gabs"
)

const DOC = `db-client.

Usage:
  db-client <configFile>
  db-client -h | --help
  db-client -v | --version

Options:
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

	//client.Debug = true

	if nil != err {
		panic(err)
	}

	go func() {
		client.MessagingLoop()
	}()

	s := bufio.NewScanner(os.Stdin)

	for {
		if ! s.Scan() {
			break
		}

		if "quit" == s.Text() {
			break
		} else if o, err := gabs.ParseJSON(s.Bytes()); nil == err {
			client.ReportState(o)
			//client.Report(newReport)
		}
	}

}
