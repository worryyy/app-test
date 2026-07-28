MODULE := github.com/Milchstrassse/Ecampus-go

.PHONY: proto-agent migrate-up migrate-down migrate-version
proto-agent:
	PATH="$(shell go env GOPATH)/bin:$$PATH"; export PATH; protoc \
		--proto_path=. \
		--go_out=. \
		--go_opt=module=$(MODULE) \
		--go-grpc_out=. \
		--go-grpc_opt=module=$(MODULE) \
		proto/agent/v1/agent.proto

migrate-up:
	go run ./cmd/ecampus-migrate up

migrate-down:
	go run ./cmd/ecampus-migrate down

migrate-version:
	go run ./cmd/ecampus-migrate version
