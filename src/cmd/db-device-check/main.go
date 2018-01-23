package main

import (
	"github.com/docopt/docopt.go"
	log "github.com/sirupsen/logrus"
	"fmt"
	"os"
	"strings"
	"gopkg.in/resty.v0"
)

const DOC = `db-device-check.

Usage:
  db-device-check [options]
  db-device-check -h | --help
  db-device-check -v | --version

Options:
  -k --api-key APIKEY     API Key to Use
  -e --endpoint ENDPOINT  API Endpoint to use [default: https://api-devices.dobkaera.cc]
  -h --help               This message
  -v --version            Shows version
`

func main() {
	opts, _ := docopt.Parse(DOC, nil, true, "0.0.1", true, true)

	//fmt.Printf("opts: %+v\n", opts)

	apiKey := os.Getenv("DB_API_KEY")

	if newApiKey, ok := opts["--api-key"].(string); ok {
		apiKey = newApiKey
	}

	if "" == apiKey {
		panic(fmt.Errorf("API Key not set!"))
	}

	endpoint := opts["--endpoint"].(string)

	if ! strings.HasSuffix(endpoint, "/") {
		endpoint += "/"
	}

	if "" == endpoint {
		panic(fmt.Errorf("Endpoint not set!"))
	}

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
