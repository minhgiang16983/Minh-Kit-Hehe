# Minh Kit

Go toolkit để scaffold microservice (gRPC + HTTP REST) với template và code generator.

## Cài đặt

```bash
cd Minh-Kit-HeHE   # hoặc thư mục minh-kit của bạn
make install       # hoặc: go install .
```

Kiểm tra:

```bash
minh-kit --help
```

## Tạo dự án mới

```bash
minh-kit new my-service
cd my-service
minh-kit gen
go mod tidy
```

Khi phát triển local (chưa push kit lên Git), thêm vào `go.mod` của service:

```go
replace github.com/minhgiang16983/Minh-Kit-Hehe => ../Minh-Kit-HeHE
```

## Lệnh có sẵn

- `minh-kit --help` — xem tất cả lệnh
- `minh-kit new <service-name>` — tạo scaffold service mới
- `minh-kit implement` — generate gRPC/REST handlers và interfaces
- `minh-kit model <sql-file>` — generate models và stores từ SQL schema
- `minh-kit gen` — generate Go code và Swagger từ `.proto`
- `minh-kit clean` — xóa file protobuf đã generate

## Generate code trong service

```bash
minh-kit model schema.sql
minh-kit implement
minh-kit gen
minh-kit clean
```
