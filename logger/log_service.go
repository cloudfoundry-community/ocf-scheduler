package logger

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

type LogService struct {
	log *logrus.Logger
}

func New() *LogService {
	log := logrus.New()
	log.Formatter = &Formatter{}

	log.Out = os.Stdout

	// Set log level from environment variable
	level, exists := os.LookupEnv("LOG_LEVEL")
	if !exists {
		log.WithField("context", "log-service").Info("No LOG_LEVEL provided. defaulting to info level")
		level = "info"
		log.SetLevel(logrus.InfoLevel)
	} else {
		switch strings.ToLower(level) {
		case "trace":
			log.SetLevel(logrus.TraceLevel)
		case "debug":
			log.SetLevel(logrus.DebugLevel)
		case "info":
			log.SetLevel(logrus.InfoLevel)
		case "warn", "warning":
			log.SetLevel(logrus.WarnLevel)
		case "error":
			log.SetLevel(logrus.ErrorLevel)
		case "fatal":
			log.SetLevel(logrus.FatalLevel)
		case "panic":
			log.SetLevel(logrus.PanicLevel)
		default:
			log.WithField("context", "log-service").Warn(fmt.Sprintf("Invalid LOG_LEVEL value '%s' , defaulting to info level", level))
			level = "info"
			log.SetLevel(logrus.InfoLevel) // Default to Info level
		}
	}
	log.WithField("context", "log-service").Info(fmt.Sprintf("Logging at %s level", level))

	return &LogService{log: log}
}

func (service *LogService) Trace(context string, message string) {
	service.log.WithField("context", context).Trace(message)
}

func (service *LogService) Debug(context string, message string) {
	service.log.WithField("context", context).Debug(message)
}

func (service *LogService) Info(context string, message string) {
	service.log.WithField("context", context).Info(message)
}

func (service *LogService) Warn(context string, message string) {
	service.log.WithField("context", context).Warn(message)
}

func (service *LogService) Error(context string, message string) {
	service.log.WithField("context", context).Error(message)
}

func (service *LogService) Fatal(context string, message string) {
	service.log.WithField("context", context).Fatal(message)
}

func (service *LogService) Panic(context string, message string) {
	service.log.WithField("context", context).Panic(message)
}
