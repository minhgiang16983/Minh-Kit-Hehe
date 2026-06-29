# 🚀 User management Service (Go + gRPC + HTTP)

A lightweight, scalable user management microservice built in Golang. It provides both **gRPC** and **RESTful HTTP** APIs using Protocol Buffers (`.proto`) to define service

## 🛠️ Tech Stack

- [Golang](https://golang.org/)
- [gRPC](https://grpc.io/)
- [Protocol Buffers (protoc)](https://developers.google.com/protocol-buffers)
- [Go HTTP server (net/http)](https://pkg.go.dev/net/http)
- [gRPC Gateway (optional)](https://github.com/grpc-ecosystem/grpc-gateway) to expose HTTP/JSON from gRPC



## 📦 Getting Started

### 1. Install Required Tools

- Go 1.24+
- `protoc`
- Cmake
- gRPC plugin:

```bash
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
go install google.golang.org/proto/cmd/protoc-gen-go
```
Make sure $GOPATH/bin is in your PATH.

### 2. Generate Code from .proto
```bash
make gen
```

### 3. Clean generated file
```bash
make clean
```