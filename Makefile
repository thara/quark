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

quark:
	go build -ldflags '-X quark.Version=$(QUARK_VERSION)' .
