module github.com/supernetes/supernetes/config

go 1.24.0

toolchain go1.25.1

require (
	github.com/lithammer/dedent v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.10.1
	github.com/spf13/pflag v1.0.10
	github.com/supernetes/supernetes/common v0.0.0
	k8s.io/api v0.34.1
	k8s.io/apimachinery v0.34.1
	sigs.k8s.io/yaml v1.6.0
)

require (
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/zerologr v1.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	github.com/virtual-kubelet/virtual-kubelet v1.11.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/client-go v0.34.1 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/utils v0.0.0-20250604170112-4c0f3b243397 // indirect
	sigs.k8s.io/json v0.0.0-20241014173422-cfa47c3a1cc8 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.0 // indirect
)

replace github.com/supernetes/supernetes/common => ../common
