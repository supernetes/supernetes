all: client server

docker-%:
	docker build -t supernetes-build .
	docker run --rm --init $(if $(strip $(CI)),,-it) \
		-e CGO_ENABLED=0 \
		-e GOBIN=/build/bin \
		-v supernetes-build-cache:/go \
		-v .:/build \
		-w /build \
		supernetes-build \
		$(MAKE) _$*

# Internal Docker-invoked targets
%.pb.go %_grpc.pb.go: %.proto
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative $*.proto

_client _server: _%: api/helloworld.pb.go
	(cd ./$* && go install)

_proto: $(patsubst %.proto,%.pb.go,$(wildcard api/*.proto))

# go mod tidy depends on the protobuf artifacts being compiled
_tidy: _proto
	find . -mindepth 2 -type f -name go.mod -execdir go mod tidy \;

_clean:
	rm -f api/*.pb.go

# Developer API
client server proto tidy clean: %: docker-%

clean:
	docker rmi -f supernetes-build
	docker volume rm -f supernetes-build-cache

.PHONY: all client server proto tidy clean
