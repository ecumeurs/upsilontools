package logger

import (
	"log"
	"os"

	"github.com/sirupsen/logrus"
)

func InitConsole() {
	logrus.SetFormatter(&logrus.TextFormatter{})
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(os.Stdout)
}

func InitFile(file string) {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(f)
}

func InitSubLogger() *logrus.Logger {
	l := logrus.New()

	l.SetFormatter(&logrus.TextFormatter{})
	l.SetLevel(logrus.DebugLevel)
	l.SetOutput(os.Stdout)
	return l
}

func InitSubFile(file string) *logrus.Logger {
	l := logrus.New()
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	l.SetFormatter(&logrus.JSONFormatter{})
	l.SetLevel(logrus.DebugLevel)
	l.SetOutput(f)
	return l
}
