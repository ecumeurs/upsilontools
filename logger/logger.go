package logger

import (
	"log"
	"os"

	"github.com/sirupsen/logrus"
)

// InitConsole initializes the global logrus logger to output to the console.
// It uses a text formatter and sets the log level to Debug.
// @spec-link [[mechanic_mech_logger_initialization]]
func InitConsole() {
	logrus.SetFormatter(&logrus.TextFormatter{})
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(os.Stdout)
}

// InitFile initializes the global logrus logger to output to a specific file.
// It uses a JSON formatter for machine-readable logs and appends to the file.
// If the file cannot be opened, it triggers a fatal error.
// @spec-link [[mechanic_mech_logger_initialization]]
func InitFile(file string) {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(f)
}

// InitSubLogger creates and returns a new independent logrus instance for console logging.
// This is required to provide isolated logging contexts for sub-components, ensuring that
// local fields or log levels do not interfere with the global logger's state. 
// It uses a TextFormatter suitable for human readability and defaults to Stdout.
// Intent: Standardize sub-component observability while maintaining strict isolation
// of global state, which is crucial for multi-tenant or multi-service architectures.
// @spec-link [[mechanic_mech_logger_initialization]]
func InitSubLogger() *logrus.Logger {
	l := logrus.New()

	l.SetFormatter(&logrus.TextFormatter{})
	l.SetLevel(logrus.DebugLevel)
	l.SetOutput(os.Stdout)
	return l
}

// InitSubFile creates and returns a new independent logrus instance for file logging.
// It uses JSON formatting and appends to the specified file path, which is critical
// for services that need to maintain separate audit logs or high-volume diagnostic
// data away from the main console stream. It ensures machine-readability of local logs,
// making them suitable for log aggregation tools or downstream analysis pipelines.
// Intent: Provide durable, structured, and isolated logging for persistent sub-tasks.
// @spec-link [[mechanic_mech_logger_initialization]]
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
