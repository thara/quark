.PHONY: protoc
protoc:
	@rm -rf ./proto
	@protoc -I=. --go_out=. command.proto
