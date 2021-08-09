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
	rm -f proto/**/*.pb.go

.PHONY: fmt
fmt:
	go fmt $$(go list ./...)
	clang-format -i proto/*.proto
	clang-format -i proto/**/*.proto

.PHONY: lint
lint:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck $$(go list ./... | grep -v 'proto')

.PHONY: test
test:
	go clean -testcache ./...
	go test -cover -race -coverprofile=coverage.out -covermode=atomic -v $$(go list ./...)

.PHONY: proto_all
protocall:
	protoc --go_out=.. --go-grpc_out=.. proto/**/*.proto

bin/gameserver: protocall $(wildcard gameserver/**/*.go)
	go build -o $@ ./gameserver

sample_chatclient: gameserver
	go run ./example/chatclient/main.go

# quark:
# 	go build -ldflags '-X quark.Version=$(QUARK_VERSION)' .
