# Supernetes

[![Go Report Card](https://goreportcard.com/badge/github.com/supernetes/supernetes)](https://goreportcard.com/report/github.com/supernetes/supernetes)
[![Latest Release](https://img.shields.io/github/v/release/supernetes/supernetes?sort=semver)](https://github.com/supernetes/supernetes/releases)

Supernetes is an HPC bridge for your Kubernetes environment: Expose your HPC nodes to Kubernetes-native tooling and schedulers, abstracting away the HPC-specific environment.

## Building

The build system requires (rootless) `docker` (or Podman symlinked to `docker`) and `make` to be present on your system. To build the client and server binaries, simply run

```shell
make client server
```

To clean up any build artifacts, run

```shell
make clean
```

## Usage

> TODO

## TODO

- [ ] Connection from HPC client to K8s server
- [ ] Container image packaging with GHCR
- [ ] Kustomization for deployment with Flux
- [ ] API for fetching nodes from HPC environment
- [ ] Implement [Virtual Kubelet `node` API](https://pkg.go.dev/github.com/virtual-kubelet/virtual-kubelet/node)
- [ ] Supernetes Operator

## Authors

- Dennis Marttinen ([@twelho](https://github.com/twelho))

## License

[MPL-2.0](https://spdx.org/licenses/MPL-2.0.html) ([LICENSE](LICENSE))
