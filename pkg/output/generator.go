package output

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// =======================================================================================

type textEntry struct {
	Key    string `yaml:"key"`
	Format string `yaml:"format"`
}

type outputConfig struct {
	Title string `yaml:"title"`
	Intro string `yaml:"intro"`
	Error string `yaml:"error"`
	Text []textEntry `yaml:"text"`
}

type outputGeneratorImpl struct {
	config outputConfig
}

// Generator for creating the alexa message
type OutputGenerator interface {
	// Get title for the message
    Title() string

    // Prepare output message
    OutputMessage(data map[string]interface{}) string

    // Error message
	ErrorMessage() string
}

func NewOutputGenerator(vConfig *viper.Viper) (OutputGenerator, error) {
	var config outputConfig
	err := vConfig.Unmarshal(&config)
	if err != nil {
		return nil, err
	}
	return outputGeneratorImpl{
		config: config,
	}, nil
}

func (o outputGeneratorImpl) ErrorMessage() string {
	errMsg := o.config.Error
	if errMsg == "" {
		return "error"
	}
	return errMsg
}

func (o outputGeneratorImpl) Title() string {
	return o.config.Title
}

func (o outputGeneratorImpl) OutputMessage(data map[string]interface{}) string {
    out := o.config.Intro
    if len(out) > 0 {
    	out += " "
	}
	for _, textEntry := range o.config.Text {
    	key := textEntry.Key
    	format := textEntry.Format
    	if key == "" || format == "" {
    		continue
		}
		if value, ok := data[key]; ok {
			out += fmt.Sprintf(format, value)
			out += " "
		}
	}
	return strings.Trim(out, " ")
}