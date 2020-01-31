package main

import (
	"fmt"
	"os"
	"strconv"

	alexa "github.com/mikeflynn/go-alexa/skillserver"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rhuss/iot2alexa/pkg/iot2alexa"
	_ "github.com/rhuss/iot2alexa/pkg/mqtt" // for registering the backend
	"github.com/rhuss/iot2alexa/pkg/output"
)

var cfgFile string

// RootCmd starts up an alexa skill server
var RootCmd = &cobra.Command{
	Use:   "iot2alexa",
	Short: "Alexa Skill server for reporting current value from IoT devices",
	Long: `iot2alexa: Skill server for reporting IoT data obtained from an IoT backend.

Supported backends:

* mqtt - Listen on a MQTT topic and report values from the payload
	`,
	RunE: alexaRun,
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "configuration file. This argument is mandatory")
	RootCmd.PersistentFlags().Bool("log-json",false,"output own logging in JSON format")
	RootCmd.MarkFlagRequired("config")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile == "" {
		logrus.Fatalf("no configuration file given. Please add one with --config")
	}

	viper.SetConfigFile(cfgFile)
	viper.AutomaticEnv()
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		logrus.WithField("configfile", cfgFile).Fatalf("cannot read config file")
	}

	if viper.GetBool("debug") {
	 	logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
	logAsJson,_ :=  RootCmd.Flags().GetBool("log-json")
	if logAsJson {
		// Log as JSON instead of the default ASCII formatter.
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

// Command loop for running the skill server
func alexaRun(cmd *cobra.Command, args []string) error {
    skillConfig := viper.Sub("skill")

	// Validate and error out if an error occurs
	err := validateSkillConfig(skillConfig)
	if err != nil {
		return err
	}

	// Collect applications
	var applications = map[string]interface{}{}

	// Create echo application
	echoApp, err := newEchoApplication(skillConfig)
	if err != nil {
		return err
	}

	// Registry application at configured path
	applications[skillConfig.GetString("path")] = echoApp

	// Server parameters for starting up the HTTP serve
	port, verifyAwsCerts := serverParams()
	alexa.SetVerifyAWSCerts(verifyAwsCerts)
	logrus.WithField("port", port).
		WithField("verifyAwsCerts", verifyAwsCerts).
		Info("Alexa skillserver is listening")
	alexa.Run(applications, port)
	return nil
}

func serverParams() (string, bool) {
	port := viper.GetInt("server.port")
	if port == 0 {
		port = 8080
	}
	verifyTls := true
	if viper.IsSet("server.verify") {
		verifyTls = viper.GetBool("server.verify")
	}

	return strconv.Itoa(port), verifyTls
}

func newEchoApplication(config *viper.Viper) (alexa.EchoApplication, error) {
	outputGenerator, err := output.NewOutputGenerator(config.Sub("output"))
	if err != nil {
		return alexa.EchoApplication{}, err
	}
	backendHandler, err := iot2alexa.LookupBackend(config)
	if err != nil {
		return alexa.EchoApplication{}, err
	}
	err = backendHandler.Init()
	if err != nil {
		return alexa.EchoApplication{}, err
	}

	iotHandler := iot2alexa.NewAlexaHandlerFunc(backendHandler, outputGenerator)
	return alexa.EchoApplication{ // Route
		AppID:    config.GetString("appid"),
		OnIntent: iotHandler,
		OnLaunch: iotHandler,
	}, nil
}

func validateSkillConfig(skillConfig *viper.Viper) error {
	if skillConfig == nil {
		return fmt.Errorf("no 'skill:' section found in configuration %s", viper.ConfigFileUsed())
	}
	if !skillConfig.IsSet("appid") {
		return fmt.Errorf("no Alexa alexa.appId section found in configuration %s", viper.ConfigFileUsed())
	}
	if !skillConfig.IsSet("path") {
		return fmt.Errorf("no path given to listen in configuration %s", viper.ConfigFileUsed())
	}

	if !skillConfig.IsSet("output") {
		return fmt.Errorf("no output mapping defined for skill in configuration %s for obtaining IoT data", viper.ConfigFileUsed())
	}

	return nil
}
