MODULE := github.com/Milchstrassse/Ecampus-go

.PHONY: proto-agent
proto-agent:
	PATH="$(shell go env GOPATH)/bin:$$PATH"; export PATH; protoc \
		--proto_path=. \
		--go_out=. \
		--go_opt=module=$(MODULE) \
		--go-grpc_out=. \
		--go-grpc_opt=module=$(MODULE) \
		proto/agent/v1/agent.proto
