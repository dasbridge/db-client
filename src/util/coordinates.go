package util

import (
	"os"
	"fmt"
	"strings"
)

func FetchCoordinates() (apiKey, endpoint string) {
	apiKey = os.Getenv("DB_API_KEY")

	if "" == apiKey {
		panic(fmt.Errorf("API Key not set!"))
	}

	endpoint = os.Getenv("DB_ENDPOINT")

	if "" == endpoint {
		panic(fmt.Errorf("Endpoint not set!"))
	}

	if ! strings.HasSuffix(endpoint, "/") {
		endpoint += "/"
	}

	return
}
