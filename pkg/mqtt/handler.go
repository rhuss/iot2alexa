package mqtt

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/rhuss/iot2alexa/pkg/iot2alexa"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/oliveagle/jsonpath"
)

// =======================================================================================

func init() {
	iot2alexa.BackendLookups = append(iot2alexa.BackendLookups,lookupMqttBackend)
}

type mqttBackend struct {
	// configuration for the backend
	config MqttConfig
	// The last values picked up from the MQTT message
	data   map[string]interface{}
}

type AuthConfig struct {
	User string `yaml:"user"`
	Password string `yaml:"password"`
}

type MappingConfig struct {
	Key string `yaml:"key"`
	Path string `yaml:"path"`
	Scale float64 `yaml:"scale"`
	Round bool `yaml:"round"`
}

type MqttConfig struct {
	Url string `yaml:"url"`
	Host string `yaml:"host"`
	Port int `yaml:"port"`
	Topic string `yaml:"topic"`
	Auth AuthConfig `yaml:"auth"`
	Mapping []MappingConfig `yaml:"mapping"`
}

func lookupMqttBackend(vConfig *viper.Viper) (iot2alexa.BackendHandler, error) {
	var config MqttConfig
	mqttConfig := vConfig.Sub("mqtt")
	if mqttConfig == nil {
		return nil, nil
	}
	err := mqttConfig.Unmarshal(&config)
	if err != nil {
		return nil, err
	}
    err = validateConfig(&config)
	if err != nil {
		return nil, err
	}
	return mqttBackend{config: config, data: map[string]interface{}{}}, nil
}

func validateConfig(config *MqttConfig) error {
	if config.Url != "" {
		_, err := url.Parse(config.Url)
		if err != nil {
			return err
		}
	} else {
		host := config.Host
		if host == "" {
			return errors.New("no mqtt host provided in mqtt backend configuration")
		}
		port := config.Port
		if port == 0 {
			port = 1883
		}
		config.Url = fmt.Sprintf("tcp://%s:%d", host, port)
	}

	if config.Topic == "" {
		return errors.New("no topic set in mqtt backend configuration")
	}

	return nil
}

var dataMutex = &sync.Mutex{}

func (m mqttBackend) Name() string {
	return "mqtt"
}

func (m mqttBackend) Data() (map[string]interface{}, error) {
	ret := map[string]interface{}{}

	dataMutex.Lock()
	for k,v := range m.data {
		// Assuming non-ref values
		ret[k] = v
	}
	dataMutex.Unlock()
	return ret, nil
}

func (m mqttBackend) Init() error {

	opts := MQTT.NewClientOptions().AddBroker(m.config.Url)
	if m.config.Auth.User != "" {
		opts.SetUsername(m.config.Auth.User)
		opts.SetPassword(m.config.Auth.Password)

	}

	onConnectHandler, err := m.createSubscribeOnConnectHandler()
	if err != nil {
		return err
	}
	opts.
		SetClientID("iot2alexa").
		SetAutoReconnect(true).
		SetConnectionLostHandler(logConnectionLost).
		SetMaxReconnectInterval(2 * time.Second).
		SetOnConnectHandler(onConnectHandler)

	// New client
	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	// Fire up an MQTT listener
	return nil
}

func (m mqttBackend) createSubscribeOnConnectHandler() (MQTT.OnConnectHandler, error) {
	return func(client MQTT.Client) {
		// Subscribe and react on changes
		logrus.WithField("server", m.config.Url).WithField("topic", m.config.Topic).Info("Watching for MQTT message")

		if token := client.Subscribe(m.config.Topic, 0, m.newWatchHandler()); token.Wait() && token.Error() != nil {
			logrus.WithError(token.Error()).
				WithField("server", m.config.Url).
				WithField("topic", m.config.Topic).
				Error("Cannot subscribe to topic")
			// Disconnect after 1s and then let the auto reconnect kick in, hopefully to subscribe then
			client.Disconnect(1000)
		}
	}, nil
}

func (m mqttBackend) newWatchHandler() MQTT.MessageHandler {
	return func(client MQTT.Client, message MQTT.Message) {
		var payload interface{}
		json.Unmarshal([]byte(message.Payload()), &payload)
		// Lock access to shared data structure
		dataMutex.Lock()
		for _, mapping := range m.config.Mapping {
			res, err := jsonpath.JsonPathLookup(payload, mapping.Path)
			if err != nil {
				logrus.WithError(err).
					WithField("path", mapping.Path).
					WithField("message", payload).
					WithField("key", mapping.Key).
					Error("Cannot extract field from message payload")
				continue
			}
			if mapping.Scale != 0 {
				var value float64
				switch v := res.(type) {
				case float64:
					value = v * mapping.Scale
				case int:
					value = float64(v) * mapping.Scale
				default:
					logrus.WithField("value", res).Error("non-numeric field can not be scaled")
					continue
				}
				res = value
			}
			if mapping.Round {
				if value, ok := res.(float64); ok {
					m.data[mapping.Key] = int(value + 0.5)
					continue
				}
			}
			m.data[mapping.Key] = res
		}
		dataMutex.Unlock()
		logrus.WithField("data",m.data).Debug("Data update")
	}
}

func buildMappingMap(mappings []MappingConfig) map[string]MappingConfig {
	ret := map[string]MappingConfig{}
	for _, mapping := range mappings {
	    ret[mapping.Key] = mapping
	}
	return ret
}

func logConnectionLost(client MQTT.Client, err error) {
	logrus.WithError(err).Error("Lost connection")
}
