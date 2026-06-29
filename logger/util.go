package logger

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func MappingLogLevel(logLevel string) zapcore.Level {
	logLevel = strings.ToUpper(logLevel)
	switch logLevel {
	case "DEBUG":
		return zap.DebugLevel
	case "INFO":
		return zap.InfoLevel
	case "WARN":
		return zap.WarnLevel
	case "ERROR":
		return zap.ErrorLevel
	default:
		return zap.InfoLevel
	}
}

func MappingEnv(env string) string {
	env = strings.ToUpper(env)
	switch env {
	// Production
	case "P":
		return "P"
	// Development
	case "D":
		return "D"
	// Staging
	case "S":
		return "S"
	default:
		return "D"
	}
}

func MaskString(val string) string {
	l := len(val)
	if l == 0 {
		return ""
	}

	if l < 4 {
		return "****"
	}

	if l < 8 {
		return val[:2] + "****" + val[l-2:]
	}

	return val[:3] + "****" + val[l-3:]
}
