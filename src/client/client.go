package client

import (
	"types"
	"crypto/tls"
	"github.com/eclipse/paho.mqtt.golang"
	"time"
	log "github.com/sirupsen/logrus"
	"fmt"
	"os"
	"crypto/x509"
	"strings"
	builtinlog "log"
	"github.com/Jeffail/gabs"
	"sync"
	"errors"
	"encoding/json"
)

type Client struct {
	cfg           *types.DeviceConfig
	cert          tls.Certificate
	hasMqttClient bool
	mqttClient    mqtt.Client
	resultCh      chan error
	started       bool
	connected     bool
	done          chan bool
	restart       chan bool
	Debug         bool
}

func NewClient(cfg *types.DeviceConfig) (*Client, error) {
	result := &Client{
		cfg: cfg,
	}

	certToUse, err := tls.X509KeyPair([]byte(cfg.CertificatePem), []byte(cfg.PrivateKey))

	if nil != err {
		return nil, err
	}

	result.cert = certToUse

	return result, nil
}

func (c *Client) MessagingLoop() {
	var err error

	loopInterval := 30 * time.Second

	c.started = true

	defer func() {
		c.started = false
	}()

	for {
		err = c.messagingLoopInternal()

		if nil != err {
			log.Warnf("loopInternal: %s. Waiting %d seconds.", err, int(loopInterval.Seconds()))
			time.Sleep(loopInterval)
		} else {
			break
		}
	}
}

func (c *Client) onConnect(conn mqtt.Client) {
	topicsToSubscribe := map[string]byte{
		c.shadowRoot("shadow", "update", "accepted"):  1,
		c.shadowRoot("shadow", "update"):              1,
		c.shadowRoot("shadow", "update", "documents"): 1,
		c.shadowRoot("shadow", "update", "rejected"):  1,
		c.shadowRoot("shadow", "get"):                 1,
		c.shadowRoot("shadow", "get", "accepted"):     1,
		c.shadowRoot("shadow", "get", "rejected"):     1,
		c.shadowRoot("shadow", "delete"):              1,
		c.shadowRoot("shadow", "delete", "accepted"):  1,
		c.shadowRoot("shadow", "delete", "rejected"):  1,
	}

	for topicName, _ := range topicsToSubscribe {
		if token := c.mqttClient.Subscribe(topicName, 0, c.onMessage); token.Wait() || nil != token.Error() {
			if nil != token.Error() {
				log.Warnf("Error when subscribing topic '%s': %v", topicName, token.Error())

				return
			} else {
				log.Infof("Subscribed to topic: %s", topicName)
			}
		} else {
			log.Infof("Subscribed to topic: %s", topicName)
		}
	}

	log.Info("Connected")

	doc := gabs.New()

	doc.Set(true, "state", "reported", "connected")

	log.Infof("Sending status update: %s", doc.String())

	t := conn.Publish(c.shadowRoot("shadow", "update"), 1, false, doc.Bytes())

	if t.Wait(); nil != t.Error() {
		log.Warnf("Unable to update thing shadow: %v", t.Error())
	} else {
		log.Info("Thing shadow updated successfully")
	}

	c.connected = true
}

func (c *Client) messagingLoopInternal() error {
OUTER:
	for {
		connectionOptions := mqtt.NewClientOptions()

		connectionOptions.SetClientID(c.cfg.ThingName)
		connectionOptions.SetCleanSession(true)
		//connectionOptions.SetAutoReconnect(true)
		connectionOptions.SetMaxReconnectInterval(180 * time.Second)
		connectionOptions.SetKeepAlive(30 * time.Second)
		connectionOptions.SetPingTimeout(15 * time.Second)
		connectionOptions.SetProtocolVersion(4)

		if lastWillDoc := gabs.New(); nil != lastWillDoc {
			lastWillDoc.Set(false, "state", "reported", "connected")

			connectionOptions.SetBinaryWill(c.shadowRoot("shadow", "update"), lastWillDoc.Bytes(), 0, false)
		}

		connectionOptions.SetConnectionLostHandler(c.onConnectionLost)

		connectionOptions.SetTLSConfig(&tls.Config{
			Certificates: []tls.Certificate{c.cert},
			RootCAs:      c.GetRootCertificates(),
		})

		connectionOptions.SetConnectTimeout(60 * time.Second)

		connectionOptions.SetWriteTimeout(15 * time.Second)

		connectionOptions.TLSConfig.BuildNameToCertificate()

		brokerUrl := fmt.Sprintf("tls://%s:8883", c.cfg.Endpoint)
		//brokerUrl := fmt.Sprintf("ws://data.iot.us-west-2.amazonaws.com:8443/mqtt")

		connectionOptions.AddBroker(brokerUrl)

		connectionOptions.OnConnect = c.onConnect

		connectionOptions.SetDefaultPublishHandler(c.onMessage)

		if c.Debug {
			mqtt.DEBUG = builtinlog.New(os.Stderr, "DEBUG: ", builtinlog.Lshortfile)
			mqtt.ERROR = builtinlog.New(os.Stderr, "ERROR: ", builtinlog.Lshortfile)
			mqtt.CRITICAL = builtinlog.New(os.Stderr, "CRITICAL: ", builtinlog.Lshortfile)
			mqtt.WARN = builtinlog.New(os.Stderr, "WARN: ", builtinlog.Lshortfile)
		}

		c.mqttClient = mqtt.NewClient(connectionOptions)

		if token := c.mqttClient.Connect(); token.Wait() || nil != token.Error() {
			if nil != token.Error() {
				return token.Error()
			}
		}

		for {
			select {
			case <-c.restart:
				continue OUTER
			case <-c.done:
				return nil
			}
		}
	}
}

func (c *Client) Start() {
	if ! c.started {
		go func() {
			c.MessagingLoop()
		}()
	}
}

func (c *Client) Stop() {
	c.done <- true
}

func (c *Client) onConnectionLost(client mqtt.Client, err error) {
	log.Infof("onConnectionLost: %v", err)

	c.connected = false

	if !client.IsConnected() {
		time.Sleep(60 * time.Second)

		c.restart <- true
	}
}
func (c *Client) GetRootCertificates() *x509.CertPool {
	pool := x509.NewCertPool()

	for _, v := range c.cfg.RootCertificates {
		pool.AppendCertsFromPEM([]byte(v))
	}

	return pool
}

func (c *Client) shadowRoot(elements ...string) string {
	result := fmt.Sprintf("$aws/things/%s/%s", c.cfg.ThingName, strings.Join(elements, "/"))

	result = strings.TrimSuffix(result, "/")

	return result
}

func (c *Client) onMessage(client mqtt.Client, message mqtt.Message) {
	payload := make(map[string]interface{})

	err := json.Unmarshal(message.Payload(), &payload)

	if nil != err {
		log.Warn("Oops: ", err)
	}

	payload["topic"] = message.Topic()

	handled := false

	for _, v := range c.cfg.Callbacks {
		if 0 != len(v.TopicSuffix) && strings.HasSuffix(message.Topic(), v.TopicSuffix) {
			beenHandled, err := v.Handle(payload)

			handled = handled || beenHandled

			if nil != err {
				log.Warnf("Oops (topicSuffix: %s): %s", v.TopicSuffix, err)
			}
		}
	}

	reformattedPayload, _ := json.Marshal(payload)

	if !handled {
		log.Debugf("Oops: Message not dealt (topic: %s): %s", message.Topic(), string(reformattedPayload))
	}
}

var publisherMu sync.Mutex

var ENotConnected = errors.New("Not connected")

func (c *Client) ReportState(o *gabs.Container) error {
	if !c.connected {
		return ENotConnected
	}

	defer func() {
		x := recover()

		if err, errorP := x.(error); errorP {
			log.Errorf("Oops: %v", err)
		}
	}()

	publisherMu.Lock()

	defer publisherMu.Unlock()

	newReport, _ := gabs.ParseJSON([]byte(fmt.Sprintf(`{"state":{"reported":%s}}`, o.String())))

	topicToPublish := c.shadowRoot("shadow", "update")

	log.Infof("Publishing state update to %s (update: %s)", topicToPublish, newReport.String())

	if token := c.mqttClient.Publish(topicToPublish, 1, false, newReport.Bytes()); token.Wait() || nil != token.Error() {
		if nil != token.Error() {
			return token.Error()
		}
	}

	return nil
}
