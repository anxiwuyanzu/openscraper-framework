package log

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	myLogger *zap.Logger
	lock     sync.Mutex
)

func Logger(debug bool, fileName string) *zap.Logger {
	if myLogger != nil {
		return myLogger
	}

	lock.Lock()
	myLogger = NewLogger(debug, fileName)
	lock.Unlock()
	return myLogger
}

func NewLogger(debug bool, fileName string) *zap.Logger {
	if debug {
		config := zap.NewDevelopmentEncoderConfig()
		config.EncodeTime = zapcore.ISO8601TimeEncoder
		consoleEncoder := zapcore.NewConsoleEncoder(config)
		core := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zap.DebugLevel)

		return zap.New(core)
	}

	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
	})
	config := zap.NewDevelopmentEncoderConfig()
	config.EncodeTime = EpochTimeEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config),
		w,
		zap.InfoLevel,
	)
	return zap.New(core)
}

func EpochTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendInt64(t.Unix())
}

func SetupLogrus() {
	logrus.SetFormatter(&logrus.TextFormatter{
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			if f == nil {
				return "", ""
			}
			filename := ""
			slash := strings.LastIndex(f.File, "/")
			if slash >= 0 {
				filename = f.File[slash+1:]
			}
			return "", fmt.Sprintf("%s:%d", filename, f.Line)
		},
		TimestampFormat: "15:04:05",
		FullTimestamp:   true,
		ForceQuote:      true,
	})

	logrus.SetReportCaller(true)
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)
}
