package logger

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

type LogService struct {
	log *logrus.Logger
}

func New() *LogService {
	log := logrus.New()
	log.Formatter = &Formatter{}
	log.Out = os.Stdout

	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = "info"
		log.WithField("context", "log-service").Info("No LOG_LEVEL provided, defaulting to info level")
	}

	parsed, err := logrus.ParseLevel(level)
	if err != nil {
		parsed = logrus.InfoLevel
		log.WithField("context", "log-service").Warn(fmt.Sprintf("Invalid LOG_LEVEL value '%s', defaulting to info level", level))
	}
	log.SetLevel(parsed)

	log.WithField("context", "log-service").Info(fmt.Sprintf("Logging at %s level", parsed))

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
