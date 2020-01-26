package iot2alexa

import (
	"fmt"
	"log"
	"strings"

	alexa "github.com/mikeflynn/go-alexa/skillserver"
	"github.com/spf13/viper"

	"github.com/rhuss/iot2alexa/pkg/output"
)

var BackendLookups []BackendLookup

type BackendHandler interface {
	Name() string
	Init() error
	Data() (map[string]interface{}, error)
}

type BackendLookup func(config *viper.Viper) BackendHandler

func LookupBackend(config *viper.Viper) (BackendHandler, error) {
	var found []BackendHandler

	for _, lookup := range BackendLookups {
		backendHandler := lookup(config)
		if backendHandler != nil {
			found = append(found, backendHandler)
			return backendHandler, nil
		}
	}

	if len(found) == 0 {
		return nil, fmt.Errorf("no backend configuration section provided in configuration %s (Known backends: mqtt)", viper.ConfigFileUsed())
	}

	// Fail if more than one backend is found
	if len(found) > 1 {
		var configured []string
		for _,backend := range found {
			configured = append(configured, backend.Name())
		}
		return nil, fmt.Errorf("multiple backends found: %s. please configured only a single backend", strings.Join(configured, ","))
	}
	return found[0], nil
}



type AlexaHandlerFunc func(echoReq *alexa.EchoRequest, echoResp *alexa.EchoResponse)

func NewAlexaHandlerFunc(handler BackendHandler, outputGenerator output.OutputGenerator) AlexaHandlerFunc {
	return func(echoReq *alexa.EchoRequest, echoResp *alexa.EchoResponse)  {
		data, err := handler.Data()
		var msg string
		if err != nil {
			msg = outputGenerator.ErrorMessage()
		} else {
			msg = outputGenerator.OutputMessage(data)
		}
		log.Printf("Output: %s\n", msg)
		echoResp.OutputSpeech(msg).Card(outputGenerator.Title(), msg)
	}
}

