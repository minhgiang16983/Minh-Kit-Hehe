package mock

import (
	"context"
	"fmt"
	"time"

	"github.com/minhgiang16983/Minh-Kit-Hehe/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type MockLogger struct{}

func NewMockLogger() logger.LoggerInterface {
	return &MockLogger{}
}

func (l *MockLogger) Info(msg string, fields ...zap.Field) {}

func (l *MockLogger) Error(msg string, fields ...zap.Field) {}

func (l *MockLogger) Debug(msg string, fields ...zap.Field) {}

func (l *MockLogger) Warn(msg string, fields ...zap.Field) {}

func (l *MockLogger) Fatal(msg string, fields ...zap.Field) {}

func (l *MockLogger) Panic(msg string, fields ...zap.Field) {}

func (l *MockLogger) String(key string, value string) zap.Field {
	return zap.String(key, value)
}

func (l *MockLogger) Int(key string, value int) zap.Field {
	return zap.Int(key, value)
}

func (l *MockLogger) Int64(key string, value int64) zap.Field {
	return zap.Int64(key, value)
}

func (l *MockLogger) Int32(key string, value int32) zap.Field {
	return zap.Int32(key, value)
}

func (l *MockLogger) Int16(key string, value int16) zap.Field {
	return zap.Int16(key, value)
}

func (l *MockLogger) Int8(key string, value int8) zap.Field {
	return zap.Int8(key, value)
}

func (l *MockLogger) Uint(key string, value uint) zap.Field {
	return zap.Uint(key, value)
}

func (l *MockLogger) Uint64(key string, value uint64) zap.Field {
	return zap.Uint64(key, value)
}

func (l *MockLogger) Uint32(key string, value uint32) zap.Field {
	return zap.Uint32(key, value)
}

func (l *MockLogger) Uint16(key string, value uint16) zap.Field {
	return zap.Uint16(key, value)
}

func (l *MockLogger) Uint8(key string, value uint8) zap.Field {
	return zap.Uint8(key, value)
}

func (l *MockLogger) Float32(key string, value float32) zap.Field {
	return zap.Float32(key, value)
}

func (l *MockLogger) Float64(key string, value float64) zap.Field {
	return zap.Float64(key, value)
}

func (l *MockLogger) Bool(key string, value bool) zap.Field {
	return zap.Bool(key, value)
}

func (l *MockLogger) Duration(key string, value time.Duration) zap.Field {
	return zap.Duration(key, value)
}

func (l *MockLogger) Time(key string, value time.Time) zap.Field {
	return zap.Time(key, value)
}

func (l *MockLogger) Any(key string, value interface{}) zap.Field {
	return zap.Any(key, value)
}

func (l *MockLogger) Object(key string, value zapcore.ObjectMarshaler) zap.Field {
	return zap.Object(key, value)
}

func (l *MockLogger) ErrorField(err error) zap.Field {
	return zap.Error(err)
}

func (l *MockLogger) With(_ ...zapcore.Field) logger.LoggerInterface {
	return l
}

func (l *MockLogger) Print(v ...interface{}) {
	l.Info(fmt.Sprint(v...))
}

func (l *MockLogger) Printf(format string, v ...interface{}) {
	l.Info(fmt.Sprintf(format, v...))
}

func (l *MockLogger) Println(v ...interface{}) {
	l.Info(fmt.Sprint(v...))
}

func (l *MockLogger) MaskString(k string, v string) zap.Field {
	return zap.String(k, v)
}

func (l *MockLogger) WithContext(_ context.Context) logger.LoggerInterface {
	return l
}
