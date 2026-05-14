.PHONY: generate build test lint

# Regenerate proto-derived files using easyp.
# Requires: easyp (go install github.com/easyp-tech/easyp/cmd/easyp@latest)
#           protoc-gen-go (go install google.golang.org/protobuf/cmd/protoc-gen-go@latest)
#           protoc-gen-mcp (go install github.com/easyp-tech/protoc-gen-mcp/cmd/protoc-gen-mcp@v0.5.0)
generate:
	easyp mod download
	easyp generate

build:
	go build ./...

test:
	go test ./...

lint:
	go vet ./...
