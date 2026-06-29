package logger

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/minhgiang16983/Minh-Kit-Hehe/attributes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LoggerInterface interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
	Panic(msg string, fields ...zap.Field)
	String(key string, value string) zap.Field
	Bool(key string, value bool) zap.Field
	Int(key string, value int) zap.Field
	Int32(key string, value int32) zap.Field
	Int64(key string, value int64) zap.Field
	Int8(key string, value int8) zap.Field
	Int16(key string, value int16) zap.Field
	Uint8(key string, value uint8) zap.Field
	Uint16(key string, value uint16) zap.Field
	Uint(key string, value uint) zap.Field
	Uint32(key string, value uint32) zap.Field
	Uint64(key string, value uint64) zap.Field
	Float32(key string, value float32) zap.Field
	Float64(key string, value float64) zap.Field
	Time(key string, value time.Time) zap.Field
	Duration(key string, value time.Duration) zap.Field
	Any(key string, value interface{}) zap.Field
	Object(key string, value zapcore.ObjectMarshaler) zap.Field
	With(fields ...zapcore.Field) LoggerInterface
	ErrorField(err error) zap.Field
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})
	MaskString(key string, value string) zap.Field
	WithContext(ctx context.Context) LoggerInterface
}

type Logger struct {
	*zap.Logger
}

var (
	logger LoggerInterface = nil
)

func init() {
	logger = New()
}

func getSpanFromContext(ctx context.Context) (tracingId, spanId string, ok bool) {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return "", "", false
	}

	if span.SpanContext().HasSpanID() {
		spanId = span.SpanContext().SpanID().String()
	}

	if span.SpanContext().HasTraceID() {
		tracingId = span.SpanContext().TraceID().String()
	}

	return tracingId, spanId, true
}

func InjectAttributes(ctx context.Context, lg LoggerInterface) LoggerInterface {
	// get request id from context
	attribute, ok := attributes.GetFromContext(ctx)
	if !ok {
		return lg
	}

	var fields []zap.Field

	if attribute.RequestID != "" {
		fields = append(fields, zap.String("x_request_id", attribute.RequestID))
	}

	if attribute.UserID != "" {
		fields = append(fields, zap.String("x_user_id", attribute.UserID))
	}

	if attribute.StaffID != "" {
		fields = append(fields, zap.String("x_staff_id", attribute.StaffID))
	}

	if attribute.DeviceID != "" {
		fields = append(fields, zap.String("x_device_id", attribute.DeviceID))
	}

	tracingId, spanId, ok := getSpanFromContext(ctx)
	if !ok {
		if tracingId != "" {
			fields = append(fields, zap.String("x_trace_id", tracingId))
			fields = append(fields, zap.String("x_span_id", spanId))
		}
	}

	newLg := lg.With(fields...)
	return newLg
}

func GetLogger(ctx context.Context) LoggerInterface {
	return InjectAttributes(ctx, logger)

}

func New() LoggerInterface {
	// Get env
	// env := os.Getenv("ENVIRONMENT")
	// env = MappingEnv(env)

	logLevel := os.Getenv("LOG_LEVEL")
	zapLogLevel := MappingLogLevel(logLevel)

	logEncoding := os.Getenv("LOG_ENCODING")
	if logEncoding == "" {
		logEncoding = "json"
	}

	encodeConfig := zap.NewProductionEncoderConfig()

	encodeConfig.TimeKey = "timestamp"
	encodeConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	config := zap.Config{
		Level:             zap.NewAtomicLevelAt(zapLogLevel),
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		Sampling:          nil,
		Encoding:          logEncoding,
		EncoderConfig:     encodeConfig,
		OutputPaths: []string{
			"stderr",
		},
		ErrorOutputPaths: []string{
			"stderr",
		},
	}

	logger := zap.Must(config.Build())

	return &Logger{Logger: logger}
}

func (l *Logger) String(key string, value string) zap.Field {
	return zap.String(key, value)
}

func (l *Logger) Int(key string, value int) zap.Field {
	return zap.Int(key, value)
}

func (l *Logger) Int64(key string, value int64) zap.Field {
	return zap.Int64(key, value)
}

func (l *Logger) Int32(key string, value int32) zap.Field {
	return zap.Int32(key, value)
}

func (l *Logger) Int16(key string, value int16) zap.Field {
	return zap.Int16(key, value)
}

func (l *Logger) Int8(key string, value int8) zap.Field {
	return zap.Int8(key, value)
}

func (l *Logger) Uint(key string, value uint) zap.Field {
	return zap.Uint(key, value)
}

func (l *Logger) Uint64(key string, value uint64) zap.Field {
	return zap.Uint64(key, value)
}

func (l *Logger) Uint32(key string, value uint32) zap.Field {
	return zap.Uint32(key, value)
}

func (l *Logger) Uint16(key string, value uint16) zap.Field {
	return zap.Uint16(key, value)
}

func (l *Logger) Uint8(key string, value uint8) zap.Field {
	return zap.Uint8(key, value)
}

func (l *Logger) Float32(key string, value float32) zap.Field {
	return zap.Float32(key, value)
}

func (l *Logger) Float64(key string, value float64) zap.Field {
	return zap.Float64(key, value)
}

func (l *Logger) Bool(key string, value bool) zap.Field {
	return zap.Bool(key, value)
}

func (l *Logger) Duration(key string, value time.Duration) zap.Field {
	return zap.Duration(key, value)
}

func (l *Logger) Time(key string, value time.Time) zap.Field {
	return zap.Time(key, value)
}

func (l *Logger) Any(key string, value interface{}) zap.Field {
	return zap.Any(key, value)
}

func (l *Logger) Object(key string, value zapcore.ObjectMarshaler) zap.Field {
	return zap.Object(key, value)
}

func (l *Logger) ErrorField(err error) zap.Field {
	return zap.Error(err)
}

func (l *Logger) With(fields ...zapcore.Field) LoggerInterface {
	base := l.Logger.With(fields...)
	return &Logger{
		Logger: base,
	}
}

func (l *Logger) WithContext(ctx context.Context) LoggerInterface {
	return InjectAttributes(ctx, l)
}

func (l *Logger) Print(v ...interface{}) {
	l.Info(fmt.Sprint(v...))
}

func (l *Logger) Printf(format string, v ...interface{}) {
	l.Info(fmt.Sprintf(format, v...))
}

func (l *Logger) Println(v ...interface{}) {
	l.Info(fmt.Sprint(v...))
}

func (l *Logger) MaskString(key string, value string) zap.Field {
	return zap.String(key, MaskString(value))
}
