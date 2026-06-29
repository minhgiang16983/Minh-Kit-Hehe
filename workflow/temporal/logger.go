package temporal

import (
	"fmt"

	"github.com/minhgiang16983/Minh-Kit-Hehe/logger"
	"go.uber.org/zap"
)

type TemporalLoggerInterface interface {
	Debug(msg string, keyVals ...interface{})
	Info(msg string, keyVals ...interface{})
	Warn(msg string, keyVals ...interface{})
	Error(msg string, keyVals ...interface{})
}
type Logger struct {
	l logger.LoggerInterface
}

func NewLogger(l logger.LoggerInterface) TemporalLoggerInterface {
	return &Logger{l: l}
}

func (s *Logger) Debug(msg string, keyVals ...interface{}) {
	s.l.Debug(msg, s.keyValsToFields(keyVals...)...)
}

func (s *Logger) Info(msg string, keyVals ...interface{}) {
	s.l.Info(msg, s.keyValsToFields(keyVals...)...)
}

func (s *Logger) Warn(msg string, keyVals ...interface{}) {
	s.l.Warn(msg, s.keyValsToFields(keyVals...)...)
}

func (s *Logger) Error(msg string, keyVals ...interface{}) {
	s.l.Error(msg, s.keyValsToFields(keyVals...)...)
}

func (s *Logger) keyValsToFields(keyVals ...interface{}) []zap.Field {
	var fields []zap.Field

	// Lặp qua từng cặp key-value
	for i := 0; i < len(keyVals); i += 2 {
		// Xử lý Key
		var key string
		if k, ok := keyVals[i].(string); ok {
			key = k
		} else {
			key = fmt.Sprintf("%v", keyVals[i])
		}

		// Xử lý Value
		var value interface{}
		if i+1 < len(keyVals) {
			value = keyVals[i+1]
		} else {
			value = "MISSING_VALUE"
		}

		fields = append(fields, s.l.Any(key, value))
	}

	return fields
}
