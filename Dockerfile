FROM golang:1-alpine

RUN apk --no-cache add findutils make protoc && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest && \
    rm -rf ~/.cache

ENTRYPOINT ["/usr/bin/make"]
