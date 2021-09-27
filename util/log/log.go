package log

import (
	"fmt"
	"github.com/assimon/captcha-bot/util/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"time"
)

var Sugar *zap.SugaredLogger

// init 日志初始化
func init() {
	writeSyncer := getLogWriter()
	encoder := getEncoder()
	core := zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel)
	logger := zap.New(core, zap.AddCaller())
	Sugar = logger.Sugar()
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}

func getLogWriter() zapcore.WriteSyncer {
	gwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	logPath := fmt.Sprintf("%s%s%s", gwd, config.TgConf.RuntimeRootPath, config.TgConf.LogSavePath)
	file := fmt.Sprintf("%s/log_%s.log",
		logPath,
		time.Now().Format("20060102"))
	lumberJackLogger := &lumberjack.Logger{
		Filename:   file,
		MaxSize:    config.TgConf.LogMaxSize,
		MaxBackups: config.TgConf.LogMaxBackups,
		MaxAge:     config.TgConf.LogMaxAge,
		Compress:   false,
	}
	return zapcore.AddSync(lumberJackLogger)
}
