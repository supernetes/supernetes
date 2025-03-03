module github.com/supernetes/supernetes/agent

go 1.23.1

require (
	al.essio.dev/pkg/shellescape v1.5.1
	github.com/jhump/grpctunnel v0.3.0
	github.com/lithammer/dedent v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.33.0
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/supernetes/supernetes/api v0.0.0
	github.com/supernetes/supernetes/common v0.0.0
	github.com/supernetes/supernetes/config v0.0.0
	google.golang.org/grpc v1.67.3
	google.golang.org/protobuf v1.36.1
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd
)

require (
	github.com/fullstorydev/grpchan v1.1.1 // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/zerologr v1.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/virtual-kubelet/virtual-kubelet v1.11.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241223144023-3abc09e42ca8 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/api v0.31.1 // indirect
	k8s.io/apimachinery v0.31.1 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/utils v0.0.0-20240711033017-18e509b52bc8 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

replace (
	github.com/supernetes/supernetes/api v0.0.0 => ../api
	github.com/supernetes/supernetes/common v0.0.0 => ../common
	github.com/supernetes/supernetes/config v0.0.0 => ../config
	k8s.io/api => k8s.io/api v0.30.0 // k8s.io/api/resource/v1alpha2 needed by k8s.io/client-go/kubernetes/scheme
)
