# Supernetes

[![Go Report Card for `agent`](https://goreportcard.com/badge/github.com/supernetes/supernetes/agent)](https://goreportcard.com/report/github.com/supernetes/supernetes/agent)
[![Go Report Card for `common`](https://goreportcard.com/badge/github.com/supernetes/supernetes/common)](https://goreportcard.com/report/github.com/supernetes/supernetes/common)
[![Go Report Card for `controller`](https://goreportcard.com/badge/github.com/supernetes/supernetes/controller)](https://goreportcard.com/report/github.com/supernetes/supernetes/controller)
[![Latest Release](https://img.shields.io/github/v/release/supernetes/supernetes?sort=semver)](https://github.com/supernetes/supernetes/releases)

Supernetes is an HPC bridge for your Kubernetes environment: Expose your HPC nodes to Kubernetes-native tooling and schedulers, abstracting away the HPC-specific environment.

## Building

The build system requires (rootless) `docker` (or Podman symlinked to `docker`), `make` and `git` to be present on your system. To build the controller and agent binaries, simply run

```shell
make
```

To build the controller as a container image, run

```shell
make image-build
```

To clean up any build artifacts, run

```shell
make clean
```

## Usage

> TODO

## TODO

- [x] Connection from HPC agent to K8s controller
  - [ ] Authentication
  - [ ] Encryption
- [x] Container image packaging with GHCR
- [x] Kustomization for deployment with Flux
- [ ] API for fetching nodes from HPC environment
- [x] Implement [Virtual Kubelet `node` API](https://pkg.go.dev/github.com/virtual-kubelet/virtual-kubelet/node)
- [ ] Implement VK pod operations
- [ ] Node pruning controller (internal or external)
- [ ] Deployment configuration

## Authors

- Dennis Marttinen ([@twelho](https://github.com/twelho))

## License

[MPL-2.0](https://spdx.org/licenses/MPL-2.0.html) ([LICENSE](LICENSE))
