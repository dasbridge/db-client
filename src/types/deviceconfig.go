package types

import (
	"os"
	"encoding/json"
	"github.com/eclipse/paho.mqtt.golang"
)

/*
    endpoint: string
    certificateId: string
    certificateArn: string
    certificatePem: string
    publicKey: string
    privateKey: string
    thingId: string
    thingArn: string
    thingName: string
    thingType: string
    thingPolicy: string
    rootCertificates: { [key: string]: string }

 */

type DeviceConfig struct {
	Endpoint         string              `json:"endpoint"`
	CertificateId    string              `json:"certificateId"`
	CertificateArn   string              `json:"certificateArn"`
	CertificatePem   string              `json:"certificatePem"`
	PublicKey        string              `json:"publicKey"`
	PrivateKey       string              `json:"privateKey"`
	ThingId          string              `json:"thingId"`
	ThingArn         string              `json:"thingArn"`
	ThingName        string              `json:"thingName"`
	ThingType        string              `json:"thingType"`
	ThingPolicy      string              `json:"thingPolicy"`
	RootCertificates map[string]string   `json:"rootCertificates"`
	PublishHandler   mqtt.MessageHandler `json:"-"`
}

func LoadDeviceConfig(path string) (*DeviceConfig, error) {
	dc := &DeviceConfig{}

	f, err := os.OpenFile(path, os.O_RDONLY, os.FileMode(0444))

	if nil != err {
		return nil, err
	}

	defer f.Close()

	err = json.NewDecoder(f).Decode(dc)

	return dc, err
}
