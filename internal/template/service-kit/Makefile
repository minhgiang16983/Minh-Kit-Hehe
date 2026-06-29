# Thư mục chứa file .proto
PROTO_DIR := ./proto

# Output cho mã Go sinh ra
OUT_PB := ./pb

# Thư mục Swagger
OUT_SWAGGER := ./docs/swagger

# Thư viện Google API (annotation)
PROTO_GOOGLEAPI := $(PROTO_DIR)/third_party

# Lấy danh sách file .proto
PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)

# Compile tất cả
gen:
	mkdir -p $(OUT_PB)
	protoc \
		-I=$(PROTO_DIR) \
		-I=$(PROTO_GOOGLEAPI) \
		--go_out=$(OUT_PB) --go_opt=paths=source_relative \
		--go-grpc_out=$(OUT_PB) --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=$(OUT_PB) --grpc-gateway_opt=paths=source_relative \
		--validate_out="lang=go,paths=source_relative:$(OUT_PB)" \
		--openapiv2_out=$(OUT_SWAGGER) --openapiv2_opt=logtostderr=true \
		$(PROTO_FILES)

# Xoá mã sinh ra
clean:
	rm -rf $(OUT_PB)/* $(OUT_SWAGGER)/*