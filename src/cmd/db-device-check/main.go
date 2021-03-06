package main

import (
	"github.com/docopt/docopt.go"
	log "github.com/sirupsen/logrus"
	"fmt"
	"gopkg.in/resty.v0"
	"util"
)

const DOC = `db-device-check.

Usage:
  db-device-check
  db-device-check -h | --help
  db-device-check -v | --version

Options:
  -h --help               This message
  -v --version            Shows version
`

func main() {
	docopt.Parse(DOC, nil, true, "0.0.1", true, true)

	//fmt.Printf("opts: %+v\n", opts)

	apiKey, endpoint := util.FetchCoordinates()

	log.Info("Using endpoint:", endpoint)

	resp, err := resty.R().
		SetHeader("x-api-key", apiKey).
		Get(endpoint)

	if nil != err {
		log.Fatalf("Oops:", err)
		panic(err)
	}

	fmt.Printf("Response: %s\n", string(resp.Body()))
}
