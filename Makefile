all: client server

_docker-%:
	docker build -t supernetes-build .
	docker run --rm --init $(if $(strip $(CI)),,-it) \
		-e CGO_ENABLED=0 \
		-e GOBIN=/build/bin \
		-e GOCACHE=/go/cache \
		-v supernetes-build-cache:/go \
		-v .:/build \
		-w /build \
		supernetes-build \
		_$*

# Internal Docker-invoked targets
%.pb.go %_grpc.pb.go: %.proto
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative $*.proto

_client _server: _%: api/supernetes.pb.go
	go install -C ./$* -ldflags "-X 'github.com/supernetes/supernetes/common/pkg/log.buildDir=$(shell pwd)'"

_proto: $(patsubst %.proto,%.pb.go,$(wildcard api/*.proto))

# go mod tidy depends on the protobuf artifacts being compiled
_tidy: _proto
	find . -type f -name go.mod -execdir go mod tidy \;

_clean:
	rm -rf bin/
	rm -f api/*.pb.go

_interactive:
	sh # Spawn an interactive shell inside the build container

# Developer API
client server proto tidy clean interactive: %: _docker-%

clean:
	docker rmi -f supernetes-build
	docker volume rm -f supernetes-build-cache

.PHONY: all client server proto tidy clean interactive
