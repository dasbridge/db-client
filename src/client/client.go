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
)

type Client struct {
	cfg           *types.DeviceConfig
	cert          tls.Certificate
	hasMqttClient bool
	mqttClient    mqtt.Client
	resultCh      chan error
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

	for {
		err = c.messagingLoopInternal()

		if nil != err {
			log.Warnf("loopInternal: %s. Waiting %d seconds.", err, int(loopInterval.Seconds()))
			time.Sleep(loopInterval)
		}
	}
}

func (c *Client) onConnect(conn mqtt.Client) {
	topicsToSubscribe := map[string]byte{
		c.shadowRoot("shadow", "update", "accepted"): 1,
		c.shadowRoot("shadow", "update"): 1,
		c.shadowRoot("shadow", "update", "documents"): 1,
		c.shadowRoot("shadow", "update", "rejected"): 1,
		c.shadowRoot("shadow", "get"): 1,
		c.shadowRoot("shadow", "get", "accepted"): 1,
		c.shadowRoot("shadow", "get", "rejected"): 1,
		c.shadowRoot("shadow", "delete"): 1,
		c.shadowRoot("shadow", "delete", "accepted"): 1,
		c.shadowRoot("shadow", "delete", "rejected"): 1,
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

		if nil == c.cfg.PublishHandler {
			connectionOptions.SetDefaultPublishHandler(c.onMessage)
		} else {
			connectionOptions.SetDefaultPublishHandler(c.cfg.PublishHandler)
		}

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
				break
				/*
			case record := <-c.recordCh:
				{
					c.publishRecord(record)
				}*/
			}
		}
	}
}

func (c *Client) Close() {
	c.done <- true
}

func (c *Client) onConnectionLost(client mqtt.Client, err error) {
	log.Infof("onConnectionLost: %v", err)

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
	//log.Infof("Topic Message Received: %s", goon.Sdump(message))

	topic := message.Topic()
	payload, err := gabs.ParseJSON(message.Payload())

	if nil != err {
		return
	}

	payload.Set(topic, "_topic")

	fmt.Println(payload.StringIndent("", "  "))
}

var publisherMu sync.Mutex

var ENotConnected = errors.New("Not connected")

func (c *Client) ReportState(o *gabs.Container) error {
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

/*
var publisherMu sync.Mutex

var ENotConnected = errors.New("Not connected")

func (c *Collector) publishRecord(record aircraft.AircraftRecord) error {
	defer func() {
		x := recover()

		if err, errorP := x.(error); errorP {
			log.Errorf("Oops: %v", err)

			c.resultCh <- err
		} else {
			c.resultCh <- nil
		}
	}()

	publisherMu.Lock()

	defer publisherMu.Unlock()

	if !c.mqttClient.IsConnected() {
		panic(ENotConnected)
	}

	buffer, err := c.encodeToRecord(record)

	if nil != err {
		panic(err)
	}

	topicToPublish := c.deviceRoot()

	if token := c.mqttClient.Publish(topicToPublish, 1, false, buffer.Bytes()); token.Wait() || nil != token.Error() {
		if nil != token.Error() {
			panic(token.Error())
		}
	}

	return nil
}

 */