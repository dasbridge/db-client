package main

import (
	"github.com/docopt/docopt.go"
	log "github.com/sirupsen/logrus"
	"fmt"
	"gopkg.in/resty.v0"
	"util"
)

const DOC = `db-device-list.

Usage:
  db-device-list [options]
  db-device-list -h | --help
  db-device-list -v | --version

Options:
  -h --help               This message
  -v --version            Shows version
`

func main() {
	docopt.Parse(DOC, nil, true, "0.0.1", true, true)

	apiKey, endpoint := util.FetchCoordinates()

	urlToUse := endpoint + "device"

	log.Infof("Using endpoint: %s (in fact: %s)", endpoint, urlToUse)

	resp, err := resty.R().
		SetHeader("x-api-key", apiKey).
		Get(urlToUse)

	if nil != err {
		log.Fatalf("Oops:", err)
		panic(err)
	}

	fmt.Printf("Response: %s (%d)\n", string(resp.Body()), resp.StatusCode())

	if 200 == resp.StatusCode() {
		fmt.Println(string(resp.Body()))
	}
}
