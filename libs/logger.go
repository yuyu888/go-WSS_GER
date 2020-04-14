package libs

import (
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	// "time"
    "wssgo/config"
)
const (
	ENV_DEV = "dev"
	ENV_PRO = "pro"
)
var (
	Logger *zap.SugaredLogger
	env string
)
func InitLogger(logName, appName, level string) {
	env = config.ServiceConf.BaseConf.Env
	var loggerLevel zapcore.Level
	switch level {
	case "debug":
		loggerLevel = zap.DebugLevel;
	case "info":
		loggerLevel = zap.InfoLevel;
	case "warn":
		loggerLevel = zap.WarnLevel;
	case "error":
		loggerLevel = zap.ErrorLevel;
	default :
		loggerLevel = zap.InfoLevel;
	}
	core := zapcore.NewCore(getEncoder(), getWriter(logName), loggerLevel)
	field := zap.Fields(zap.String("app_name", appName))
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel), field)
	Logger = zapLogger.Sugar()

}

func getEncoder() zapcore.Encoder{
	encoderConfig := zapcore.EncoderConfig{
		TimeKey	:	"time",
		LevelKey : 	"level",
		NameKey	:	"name",
		CallerKey : "line",
		MessageKey : "log",
		StacktraceKey : "trace",
		LineEnding	:	zapcore.DefaultLineEnding,
		EncodeLevel	:	zapcore.LowercaseColorLevelEncoder,
		EncodeTime	:	zapcore.ISO8601TimeEncoder,
		EncodeDuration :	zapcore.SecondsDurationEncoder,
		EncodeCaller : zapcore.FullCallerEncoder,
		EncodeName	:	zapcore.FullNameEncoder,
	}
	if env == ENV_DEV {
		return zapcore.NewConsoleEncoder(encoderConfig);
	} 
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder;
	return zapcore.NewJSONEncoder(encoderConfig);
}

func getWriter(logName string) zapcore.WriteSyncer{
	if env == ENV_PRO {
		lumber := &lumberjack.Logger{
			Filename : config.ServiceConf.BaseConf.LogDir + logName,
			MaxSize : 1,
			MaxBackups : 5,
			MaxAge : 30,
			Compress : true,
		}
		return zapcore.AddSync(lumber);
	}
	return zapcore.AddSync(os.Stdout);
}