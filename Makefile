QUARK_VERSION = 0.0.1

# Tool versoins
PROTOC_VERSION = 3.17.3
PROTOC_GEN_DOC_VERSION = 1.4.1
PROTOC_GEN_GO_VERSION = 1.26.0
PROTOC_GEN_GO_GRPC_VERSION = 1.1.0

PROTOC_GEN_GO = $(GOPATH)/bin/protoc-gen-go
PROTOC_GEN_GO_GRPC = $(PWD)/bin/protoc-gen-go-grpc

$(PROTOC_GEN_GO):
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v$(PROTOC_GEN_GO_VERSION)

$(PROTOC_GEN_GO_GRPC):
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v$(PROTOC_GEN_GO_GRPC_VERSION)

.PHONY: clean
clean:
	rm -rf bin
	rm -f health/*.pb.go
	rm -f gameserver/*.pb.go

.PHONY: fmt
fmt:
	go fmt $$(go list ./...)
	clang-format -i proto/**/*.proto

health/health.pb.go: $(PROTOC_GEN_GO)
	protoc --go_out=.. proto/health/health.proto

health/health_grpc.pb.go: $(PROTOC_GEN_GO_GRPC)
	protoc --go-grpc_out=.. proto/health/health.proto

gameserver/room.pb.go: $(PROTOC_GEN_GO)
	protoc --go_out=.. proto/gameserver/room.proto

gameserver/room_grpc.pb.go: $(PROTOC_GEN_GO_GRPC)
	protoc --go-grpc_out=.. proto/gameserver/room.proto

gameserver: gameserver/room.pb.go gameserver/room_grpc.pb.go health/health.pb.go health/health_grpc.pb.go $(wildcard gameserver/**/*.go)
	go build -o bin/$@ ./gameserver/cmd/gameserver

# quark:
# 	go build -ldflags '-X quark.Version=$(QUARK_VERSION)' .
