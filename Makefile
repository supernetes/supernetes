all: agent controller config

c := ,
_docker: # Force target for pattern rule phony
_docker-%: _docker
	docker build -t supernetes-build . $(if $(strip $(GITHUB_ACTIONS)),--cache-to type=gha$(c)mode=max --cache-from type=gha,)
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

# Trim build directory and GOPATH from paths registered in built binaries
go_flags := "all=-trimpath=$(shell pwd);$(GOPATH)"

_agent _controller: _%: src/api/v1alpha1/node.pb.go src/api/v1alpha1/workload.pb.go
	go install -C src/$* -gcflags $(go_flags) -asmflags $(go_flags)

_config: _%:
	go install -C src/$* -gcflags $(go_flags) -asmflags $(go_flags)

_proto: $(patsubst %.proto,%.pb.go,$(wildcard src/api/*/*.proto))

# go mod tidy depends on the protobuf artifacts being compiled
_tidy: _proto
	find . -type f -name go.mod -execdir go mod tidy \;

_clean:
	rm -rf bin/
	rm -f src/api/*/*.pb.go

_interactive:
	sh # Spawn an interactive shell inside the build container

IMAGE_REPO ?= ghcr.io/supernetes/controller
IMAGE_TAG ?= $(shell git describe --tags --dirty)
IMAGE := "$(IMAGE_REPO):$(IMAGE_TAG)"

# Developer API
agent controller config proto tidy clean interactive: %: _docker-%

image-build: controller
	docker build -t $(IMAGE) -f build/Dockerfile .

image-push: image-build
ifneq ($(findstring dirty,$(IMAGE_TAG)),)
	$(error will not push build from dirty working tree)
endif
	docker push $(IMAGE)

image-digest:
	$(warning retrieving digest for image: $(IMAGE))
	@docker inspect --format='{{index .RepoDigests 0}}' $(IMAGE)

clean:
	docker rmi -f supernetes-build $(shell docker images --filter=reference="$(IMAGE_REPO):*" -q)
	docker volume rm -f supernetes-build-cache

# Include custom targets if defined
-include dev/Makefile

# All directly invoked targets are phony, Make still checks dependencies for %.pb.go etc. as desired
.PHONY: all $(MAKECMDGOALS)
