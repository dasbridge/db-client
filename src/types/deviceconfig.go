package types

import (
	"os"
	"encoding/json"
	"github.com/jmespath/go-jmespath"
)

type Callback func(map[string]interface{}, interface{}) error

type CallbackDefinition struct {
	TopicSuffix      string
	JmesPath         string
	Callback         Callback
	jmesPathCompiled *jmespath.JMESPath
}

func (c *CallbackDefinition) Handle(m map[string]interface{}) (validP bool, err error) {
	val, err := c.jmesPathCompiled.Search(m)

	validP = nil != val && nil == err

	if !validP {
		return
	}

	err = c.Callback(m, val)

	return
}

func (c *CallbackDefinition) Compile() error {
	if newJMesPath, err := jmespath.Compile(c.JmesPath); nil != err {
		return err
	} else {
		c.jmesPathCompiled = newJMesPath

		return nil
	}
}

type DeviceConfig struct {
	Endpoint         string               `json:"endpoint"`
	CertificateId    string               `json:"certificateId"`
	CertificateArn   string               `json:"certificateArn"`
	CertificatePem   string               `json:"certificatePem"`
	PublicKey        string               `json:"publicKey"`
	PrivateKey       string               `json:"privateKey"`
	ThingId          string               `json:"thingId"`
	ThingArn         string               `json:"thingArn"`
	ThingName        string               `json:"thingName"`
	ThingType        string               `json:"thingType"`
	ThingPolicy      string               `json:"thingPolicy"`
	RootCertificates map[string]string    `json:"rootCertificates"`
	Callbacks        []*CallbackDefinition `json:"-"`
}

func (d *DeviceConfig) AddCallbackRule(callback *CallbackDefinition) error {
	if err := callback.Compile(); nil != err {
		return err
	}

	d.Callbacks = append(d.Callbacks, callback)

	return nil
}

func LoadDeviceConfig(path string) (*DeviceConfig, error) {
	dc := &DeviceConfig{}

	f, err := os.OpenFile(path, os.O_RDONLY, os.FileMode(0444))

	if nil != err {
		return nil, err
	}

	defer f.Close()

	err = json.NewDecoder(f).Decode(dc)

	dc.Callbacks = make([]*CallbackDefinition, 0)

	return dc, err
}
