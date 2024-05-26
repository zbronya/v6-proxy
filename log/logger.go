package log

import "github.com/sirupsen/logrus"

type Logger interface {
	Info(format string, args ...interface{})

	Warn(format string, args ...interface{})
	Fatal(format string, args ...interface{})
}

type LogrusLogger struct {
	logger *logrus.Logger
}

func (l *LogrusLogger) Info(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

func (l *LogrusLogger) Fatal(format string, args ...interface{}) {
	l.logger.Fatalf(format, args...)
}

func (l *LogrusLogger) Warn(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

func NewLogrusLogger() *LogrusLogger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	return &LogrusLogger{logger: logger}
}

var logger Logger = NewLogrusLogger()

func GetLogger() Logger {
	return logger
}
