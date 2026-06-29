# Logger Package

The `logger` package provides a structured, high-performance logging interface based on [uber-go/zap](https://github.com/uber-go/zap). It includes built-in support for context-aware logging, automatic attribute injection (Request ID, User ID, etc.), and tracing integration with OpenTelemetry.

## Features

- **Structured Logging**: Powered by `zap` for high performance.
- **Context Awareness**: Automatically extracts and logs metadata from `context.Context`.
- **Automatic Attribute Injection**: Supports injection of `x_request_id`, `x_user_id`, `x_staff_id`, `x_device_id` when using `minh-kit/attributes`.
- **Tracing Support**: Automatically injects `x_trace_id` and `x_span_id` from OpenTelemetry spans.
- **Data Masking**: Provides `MaskString` utility to hide sensitive information.
- **Customizable**: Configure log level and encoding via environment variables.

## Installation

```go
import "github.com/minhgiang16983/Minh-Kit-Hehe/logger"
```

## Configuration

The logger can be configured using the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `LOG_LEVEL` | Logging level (`DEBUG`, `INFO`, `WARN`, `ERROR`) | `INFO` |
| `LOG_ENCODING` | Log output format (`json`, `console`) | `json` |

## Usage

### Basic Usage

```go
// Get the global logger instance
lg := logger.GetLogger(ctx)

// Log with fields
lg.Info("user logged in", lg.String("username", "john_doe"))
```

### Context-Aware Logging

When you use `GetLogger(ctx)`, it automatically injects attributes stored in the context (like Request ID) and OpenTelemetry tracing information.

```go
func HandleRequest(ctx context.Context) {
    lg := logger.GetLogger(ctx)
    lg.Info("processing request") // Automatically includes trace_id, request_id, etc.
}
```

### Creating a New Logger

If you need a standalone logger instance:

```go
lg := logger.New()
lg.Info("standalone logger initialized")
```

### Data Masking

Use `MaskString` to protect sensitive data in logs:

```go
lg := logger.GetLogger(ctx)
lg.Info("user info", lg.MaskString("phone", "0912345678"))
// Output: {"phone": "091****678", ...}
```

## Logger Interface

The `LoggerInterface` defines the following methods:

- **Logging Levels**: `Debug`, `Info`, `Warn`, `Error`, `Fatal`, `Panic`.
- **Field Helpers**: `String`, `Int`, `Int64`, `Bool`, `Duration`, `Time`, `Any`, `Object`, etc.
- **Context/Fields**:
    - `With(fields ...zapcore.Field) LoggerInterface`: Returns a new logger with added fields.
    - `WithContext(ctx context.Context) LoggerInterface`: Returns a new logger with attributes from context.
- **Utilities**:
    - `MaskString(key string, value string) zap.Field`: Masks the value before logging.
    - `ErrorField(err error) zap.Field`: Helper for logging errors.
