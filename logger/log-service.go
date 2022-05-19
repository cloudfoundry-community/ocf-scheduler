package logger

import (
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

	return &LogService{log: log}
}

func (service *LogService) Info(context string, message string) {
	service.log.WithField("context", context).Info(message)
}

func (service *LogService) Error(context string, message string) {
	service.log.WithField("context", context).Error(message)
}
