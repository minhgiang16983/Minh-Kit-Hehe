FROM golang:1.24.4-alpine AS builder
WORKDIR /app

# To cache dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /app/bin/server ./cmd/server/*.go

FROM alpine:3.22.0

WORKDIR /app/
COPY --from=builder /app/bin/server /app/server

EXPOSE 9000
EXPOSE 10000

CMD ["./server"]

