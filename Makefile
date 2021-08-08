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
	rm -f proto/*.pb.go
	rm -f health/*.pb.go

.PHONY: fmt
fmt:
	go fmt $$(go list ./...)
	clang-format -i proto/*.proto

.PHONY: lint
lint:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck $$(go list ./... | grep -v 'proto')

.PHONY: lint
test:
	go clean -testcache ./...
	go test -cover -race -coverprofile=coverage.out -covermode=atomic -v $$(go list ./...)

proto/health.pb.go: $(PROTOC_GEN_GO)
	protoc --go_out=.. proto/health.proto

proto/health_grpc.pb.go: $(PROTOC_GEN_GO_GRPC)
	protoc --go-grpc_out=.. proto/health.proto

proto/room.pb.go: $(PROTOC_GEN_GO)
	protoc --go_out=.. proto/room.proto

proto/room_grpc.pb.go: $(PROTOC_GEN_GO_GRPC)
	protoc --go-grpc_out=.. proto/room.proto

gameserver: proto/room.pb.go proto/room_grpc.pb.go proto/health.pb.go proto/health_grpc.pb.go $(wildcard gameserver/**/*.go)
	go build -o bin/$@ ./gameserver

sample_chatclient: gameserver
	go run ./gameserver/example/chatclient/main.go

# quark:
# 	go build -ldflags '-X quark.Version=$(QUARK_VERSION)' .
