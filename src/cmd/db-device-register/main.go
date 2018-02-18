package main

import (
	"github.com/docopt/docopt.go"
	log "github.com/sirupsen/logrus"
	"fmt"
	"os"
	"gopkg.in/resty.v0"
	"types"
	"github.com/Jeffail/gabs"
	"util"
)

const DOC = `db-device-register.

Usage:
  db-device-register [options] <thingId> <thingType>
  db-device-register -h | --help
  db-device-register -v | --version

Options:
  -h --help               This message
  -v --version            Shows version
`

func main() {
	opts, _ := docopt.Parse(DOC, nil, true, "0.0.1", true, true)

	//fmt.Printf("opts: %+v\n", opts)

	apiKey, endpoint := util.FetchCoordinates()

	thingId := opts["<thingId>"].(string)
	thingType := opts["<thingType>"].(string)

	urlToUse := endpoint + "device"

	log.Infof("Using endpoint: %s (in fact: %s)", endpoint, urlToUse)

	result := &types.DeviceConfig{}

	resp, err := resty.R().
		SetHeader("x-api-key", apiKey).
		SetBody(map[string]string{
		"thingId":   thingId,
		"thingType": thingType,
	}).
		SetResult(result).
		Post(urlToUse)

	if nil != err {
		log.Fatalf("Oops:", err)
		panic(err)
	}

	fmt.Printf("Response: %s (%d)\n", string(resp.Body()), resp.StatusCode())

	if 200 == resp.StatusCode() && "" != result.ThingId {
		log.Infof("Response Content: %s", string(resp.Body()))

		// Validate Json just in case

		_, err := gabs.ParseJSON(resp.Body())

		if nil != err {
			panic(err)
		}

		outputFileName := result.ThingName + "-" + result.ThingId + ".json"

		log.Infof("Saving config for '%s' into file '%s'", result.ThingId, outputFileName)

		outputFile, err := os.OpenFile(outputFileName, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(0644))

		if nil != err {
			panic(err)
		}

		defer outputFile.Close()

		outputFile.Write(resp.Body())
	}
}
