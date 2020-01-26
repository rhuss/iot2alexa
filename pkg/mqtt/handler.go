package mqtt

import (
	"github.com/spf13/viper"

	"github.com/rhuss/iot2alexa/pkg/iot2alexa"
)

// =======================================================================================

func init() {
	iot2alexa.BackendLookups = append(iot2alexa.BackendLookups,lookupMqttBackend)
}

type mqttBackend struct {
	config *viper.Viper
}

func lookupMqttBackend(config *viper.Viper) iot2alexa.BackendHandler {
	mqttConfig := config.Sub("mqtt")
	if mqttConfig == nil {
		return nil
	}
	return mqttBackend{config: mqttConfig}
}

func (m mqttBackend) Name() string {
	return "mqtt"
}

func (m mqttBackend) Data() (map[string]interface{}, error) {
	// Implement logic
	return map[string]interface{}{
		"temp_collector": -5,
		"temp_low": 20,
		"temp_medium": 30,
		"temp_high": 45,
	}, nil
}

func (m mqttBackend) Init() error {
	// Fire up an MQTT listener
	return nil
}

