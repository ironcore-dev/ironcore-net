# Local Development Setup

## Requirements

* `go` >= 1.20
* `git`, `make` and `kubectl`
* [Kustomize](https://kustomize.io/)
* Access to a Kubernetes cluster ([Minikube](https://minikube.sigs.k8s.io/docs/), [kind](https://kind.sigs.k8s.io/) or a
  real cluster)

## Clone the Repository

To bring up and start locally the `ironcore-net` project for development purposes clone the repository.

```shell
git clone git@github.com:ironcore-dev/ironcore-net.git
cd ironcore-net
```

## Install cert-manager

If there is no [cert-manager](https://cert-manager.io/docs/) present in the cluster it needs to be installed.

```shell
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.2/cert-manager.yaml
```

## Setup `ironcore`

Reference: [ironcore docs](https://github.com/ironcore-dev/ironcore/blob/main/docs/development/setup.md)


## Setup `ironcore-net` 

### Setup `ironcore-net` with `kind` cluster

For local development with `kind`, a make target that builds and loads the apiserver/controller images and then applies
the manifests is available via

1. Build and apply ironcore-net apiserver and controller manager to the cluster

```shell
make kind-deploy
```

2. Build and apply apinetlet to the cluster

```shell
make kind-build-load-restart-apinetlet
make kind-apply-apinetlet
```

3. Build and apply metalnetlet to the cluster

```shell
make kind-build-load-restart-metalnetlet
make kind-apply-metalnetlet
```

### Setup `ironcore-net` with any other kubernetes cluster

For local development with any kubernetes cluster other than `kind`, a make target that applies the manifests using 
provided images is available via

1. Install ironcore-net API server and API services into the K8s cluster.

```shell
make install APISERVER_IMG=<apiserver image>
```

2. Deploy ironcore-net controller, apinetlet controller and metalnetlet controller to the K8s cluster.

```shell
make deploy CONTROLLER_MANAGER_IMG=<ironcore-net controller-manager image> APINETLET_IMG=<apinetlet image> METALNETLET_IMG=<metalnetlet image>
```

If you want to deploy any of ironcore-net controller, apinetlet controller and metalnetlet controller individually 
then use below make targets respectively

```shell
make deploy-ironcore-net CONTROLLER_MANAGER_IMG=<ironcore-net controller-manager image>
make deploy-apinetlet APINETLET_IMG=<apinetlet image>
make deploy-metalnetlet METALNETLET_IMG=<metalnetlet image>
```

**Note**: This requires the `APISERVER_IMG` (Makefile default set to `apiserver`), `CONTROLLER_MANAGER_IMG` (Makefile default set to `controller`), `APINETLET_IMG` (Makefile default set to `apinetlet`) and `METALNETLET_IMG` (Makefile default set to `metalnetlet`) to be pullable from your kubernetes
cluster.

## Cleanup

### Cleanup from `kind` cluster

```shell
make kind-delete
make kind-delete-apinetlet
make kind-delete-metalnetlet
```

### Cleanup from any other kubernetes cluster

```shell
make uninstall
make undeploy
make undeploy-apinetlet
make undeploy-metalnetlet
```