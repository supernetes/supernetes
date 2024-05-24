module github.com/supernetes/supernetes/client

go 1.22.3

require (
	github.com/goombaio/namegenerator v0.0.0-20181006234301-989e774b106e
	github.com/rs/zerolog v1.33.0
	github.com/spf13/pflag v1.0.5
	github.com/supernetes/supernetes/api v0.0.0
	github.com/supernetes/supernetes/common v0.0.0
	google.golang.org/grpc v1.64.0
)

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
)

replace (
	github.com/supernetes/supernetes/api v0.0.0 => ../api
	github.com/supernetes/supernetes/common v0.0.0 => ../common
)
