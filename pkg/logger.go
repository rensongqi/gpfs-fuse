package pkg

import (
	"fmt"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
	"gpfs-fuse/settings"
	"io"
	"os"
	"path"
	"runtime"
	"time"
)

var logger = &lumberjack.Logger{
	Filename:   fmt.Sprintf("%s/%s.log", settings.LogPath, time.Now().Format(settings.Timestamp)),
	MaxSize:    10, // megabytes
	MaxBackups: 3,
	MaxAge:     28, //days
}

func init() {
	// create log path
	_, err := os.Stat(settings.LogPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err = os.Mkdir(settings.LogPath, 0755); err != nil {
				logrus.Fatal(err, ", try sudo.")
			}
		}
	}

	// setting json logger
	logFormat := &logrus.JSONFormatter{
		TimestampFormat: settings.LoggerFormat,
		PrettyPrint:     false,
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			fileName := fmt.Sprintf("%v:%v", path.Base(frame.File), frame.Line)
			return frame.Function, fileName
		},
	}
	logrus.SetReportCaller(true)
	logrus.SetFormatter(logFormat)

	//Write logs to both files and std output
	stdOutWrite := io.Writer(os.Stdout)
	fileAndStdoutWriter := io.MultiWriter(os.Stdout, logger)
	logrus.SetOutput(stdOutWrite)
	logrus.SetOutput(fileAndStdoutWriter)

	//setting loglevel
	logrus.SetLevel(logrus.InfoLevel)
	logrus.Info("The log module was initialized successfully...")

	// setting gin log writer
	//gin.DefaultWriter = io.MultiWriter(os.Stdout, logger)
}
