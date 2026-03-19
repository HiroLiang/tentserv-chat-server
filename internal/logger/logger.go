package logger

import (
	"fmt"
	"os"

	"github.com/HiroLiang/tentserv-chat-server/internal/config"
	"go.uber.org/zap"
)

var Log *zap.Logger

func Init() {
	var err error
	if config.Env("APP_ENV", "dev") == "dev" {
		Log, err = zap.NewDevelopment()
	} else {
		Log, err = zap.NewProduction()
	}
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Logger init error: %v\n", err)
		panic(err)
	}
}

func InitTestEnv() {
	var err error
	Log, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(Log)
}

func Stop() {
	if Log != nil {
		_ = Log.Sync()
	}
}
