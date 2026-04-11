package core

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// Log is the global logger instance
var Log = logrus.New()

// InitLogger initializes the global logger with a text formatter and optional file output
func InitLogger(logFilePath string) error {
	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     logFilePath == "", // Force colors if only logging to terminal
	})

	mw := io.MultiWriter(os.Stdout)

	if logFilePath != "" {
		file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		mw = io.MultiWriter(os.Stdout, file)
	}

	Log.SetOutput(mw)
	Log.SetLevel(logrus.InfoLevel)

	return nil
}
