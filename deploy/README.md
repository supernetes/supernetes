# Deploy Supernetes using Kustomize

For the Kubernetes side, [Supernetes](https://github.com/supernetes/supernetes) relies on [Kustomize](https://kustomize.io/) for deployment with total configuration freedom.

In order to directly deploy the manifests from this repository, create a deployment directory for yourself (anywhere, you do not need to clone this repository):

```shell
mkdir deployment
touch deployment/kustomization.yaml
```

Configure `deployment/kustomization.yaml` as follows:

```yaml
resources:
  - https://github.com/supernetes/deploy?ref=$BRANCH_NAME
  
# Use patches to further configure the deployment
#patches:
# - path: ...
```

If you make the `deployment` directory a Git repository, it can then be fed into a GitOps tool, such as [FluxCD](https://fluxcd.io/) or [Argo CD](https://argoproj.github.io/cd/). An example configuration for FluxCD is provided [in the bootstrap repository](https://github.com/supernetes/bootstrap/tree/master/work/manifests/flux). Alternatively, you can also directly apply the manifests into your cluster with

```shell
kubectl apply -k deployment
```
