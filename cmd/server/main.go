package main

import (
	"hanashite/internal"
	"hanashite/internal/common"

	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

func main() {
	// Setup temp logger before everything
	common.PreLogger()
	zap.S().Infof("Version %s", internal.Version())
	// Define flags
	configPath := pflag.StringP("config", "c", "./config.yml", "path to config file")
	pflag.Parse()
	common.Load(*configPath)
	var loggerConfig = zap.NewDevelopmentConfig()
	if err := common.FetchConfig("logging", &loggerConfig); err != nil {
		panic(err)
	}
	common.SetupLogger(&loggerConfig)
	defer common.ShutdownLogger()

	var serverConfig ServerConfig
	if err := common.FetchConfig("server", &serverConfig); err != nil {
		panic(err)
	}
	socketServer := NewSocketServer(&serverConfig)
	socketServer.Start()
}
