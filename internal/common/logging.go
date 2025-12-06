package common

import "go.uber.org/zap"

func PreLogger() {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
}

func SetupLogger(cfg *zap.Config) {
	logger := zap.Must(cfg.Build())
	zap.RedirectStdLog(logger)
	zap.ReplaceGlobals(logger)
}

func ShutdownLogger() {
	if err := zap.L().Sync(); err != nil {
		panic(err)
	}
}
