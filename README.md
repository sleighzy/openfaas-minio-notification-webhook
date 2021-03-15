# OpenFaaS MinIO Notification Webhook Function

This repository creates and deploys an [OpenFaaS] Golang function that is used as
a webhook by MinIO to receive notifications of events.

## Installing OpenFaaS

I'm not going to go into any great detail for installing and deploying OpenFaaS,
I'll do that as a separate set of instructions later on. I essentially followed
the directions from [OpenFaas Deployment] and used the awesome [Arkade] CLI
installer for Kubernetes applications, plus some of the linked blog posts.

## Private Docker Registry

When deploying functions from a private registry OpenFaaS needs the credentials
to be able to authenticate to it when pulling images. See [Use a private
registry with Kubernetes] for more information on this.

Run the below command to create the Docker registry credentials secret in the
`openfaas-fn` namespace.

```sh
kubectl create secret docker-registry homelab-docker-registry \
  --docker-username=homelab-user \
  --docker-password=homelab-password \
  --docker-email=homelab@example.com \
  --docker-server=https://registry.mydomain.io \
  --namespace openfaas-fn
```

Add the below yaml to the `default` service account in the `openfaas-fn`
namespace so that it has the credentials to authenticate with the registry when
pulling images.

```sh
kubectl edit serviceaccount default -n openfaas-fn
```

```yaml
imagePullSecrets:
  - name: homelab-docker-registry
```

## Creating the Function

The below steps were followed to create a new function and handler.

Run the command below to pull the `golang-http` template that creates an HTTP
request handler for Golang.

```sh
faas-cli template store pull golang-http
```

Run the command below to create the function definition files and empty function
handler.

```sh
$ faas new --lang golang-http minio-notification-webhook
Folder: minio-notification-webhook created.
  ___                   _____           ____
 / _ \ _ __   ___ _ __ |  ___|_ _  __ _/ ___|
| | | | '_ \ / _ \ '_ \| |_ / _` |/ _` \___ \
| |_| | |_) |  __/ | | |  _| (_| | (_| |___) |
 \___/| .__/ \___|_| |_|_|  \__,_|\__,_|____/
      |_|


Function created in folder: minio-notification-webhook
Stack file written: minio-notification-webhook.yml
```

### Golang Dependencies

This function uses additional Go libraries that need to be included as
dependencies when building. See [GO - Dependencies] for options on including
these dependencies. This repository uses [Go Modules] for managing dependencies.

The below commands were run to initialize the `go.mod` and `go.sum` files, and
the contents of the `go.mod` file put in the `GO_REPLACE.txt` file to be used
during the build. These commands need to be run from within the `slack`
directory containing the function handler.

```sh
$ cd minio-notification-webhook
$ export GO111MODULE=on

$ go mod init
go: creating new go.mod: module openfaas/openfaas-minio-notification-webhook/minio-notification-webhook

$ go get
go: finding module for package github.com/openfaas/templates-sdk/go-http
go: found github.com/openfaas/templates-sdk/go-http in github.com/openfaas/templates-sdk v0.0.0-20200723110415-a699ec277c12

$ go mod tidy
$ cat go.mod > GO_REPLACE.txt
```

When adding new libraries within your handler source code you will need to
update your Go dependencies.

```sh
cd minio-notification-webhook
go mod tidy
cat go.mod > GO_REPLACE.txt
```

[arkade]: https://github.com/alexellis/arkade
[docker buildx]:
  https://docs.docker.com/engine/reference/commandline/buildx_build/
[go - dependencies]: https://docs.openfaas.com/cli/templates/#go-go-dependencies
[go modules]: https://golang.org/ref/mod
[httpie]: https://httpie.io/
[openfaas]: https://www.openfaas.com/
[openfaas deployment]: https://docs.openfaas.com/deployment/
[openfaas using secrets]: https://docs.openfaas.com/reference/secrets/
[simple serverless with golang functions and microservices]:
  https://www.openfaas.com/blog/golang-serverless/
[use a private registry with kubernetes]:
  https://docs.openfaas.com/deployment/kubernetes/#use-a-private-registry-with-kubernetes
