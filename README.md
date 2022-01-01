# OpenFaaS MinIO Notifications Webhook Function

[MinIO] is an open source object storage server with support for the S3 API.
This means that you can run your very own S3 deployment from your homelab.

This repository creates and deploys an [OpenFaaS] Golang function that is used
as a webhook by MinIO to receive notifications of events.

See the complete [MinIO Bucket Notification Guide] documentation for events and
subscribing to them.

## OpenFaas Slack Function

Whilst you can do anything you like with your OpenFaas function handler for
these notification events, I have chosen to invoking another OpenFaaS function I
have. This is the [OpenFaaS Slack Function] that you can locate in one of my
other Github repositories. This is a simple function which takes a basic JSON
payload and uses the [slack-go] library to send messages to a Slack channel.

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

## Generating Go Type Definitions from JSON

The MinIO notification events are sent to the endpoint as a JSON payload. To be
able to marshall the payloads into Go type definitions I have used the excellent
online tool [JSON-to-Go] to generate these. By using example JSON payloads from
MinIO you can easily generate the Go code.

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

The below commands were run to initialize the `go.mod` and `go.sum` files. These
commands need to be run from within the `slack` directory containing the
function handler.

```sh
$ cd minio-notification-webhook
$ export GO111MODULE=on

$ go mod init
go: creating new go.mod: module openfaas/openfaas-minio-notification-webhook/minio-notification-webhook

$ go get
go: finding module for package github.com/openfaas/templates-sdk/go-http
go: found github.com/openfaas/templates-sdk/go-http in github.com/openfaas/templates-sdk v0.0.0-20200723110415-a699ec277c12

$ go mod tidy
```

When adding new libraries within your handler source code you will need to
update your Go dependencies.

```sh
cd minio-notification-webhook
go mod tidy
```

## Building the Function

The OpenFaaS documentation and [Simple Serverless with Golang Functions and
Microservices] provide instruction on how to develop and build OpenFaaS
functions.

### ARM64 Image Builds

This function is going to be deployed onto a Raspberry Pi using ARM64 so the
build and deploy process is slightly different than a basic `faas-cli up`
command. The below command will create a new directory containing the
`Dockerfile` and artifacts that will be used to build the container image.

```sh
faas-cli build --shrinkwrap -f minio-notification-webhook.yml
```

### Docker Buildx for multiple platforms

The below commands should only need to be run once but will create a new Docker
build context for using with [Docker Buildx] to create images for multiple
platforms.

```sh
export DOCKER_CLI_EXPERIMENTAL=enabled
docker buildx create --use --name=multiarch
docker buildx inspect --bootstrap
```

Run the below command to use Buildx to create an image that supports both amd64
and arm64 architectures, and push it to the registry. This sets the
`GO111MODULE` build arg to `on` so that the Go dependencies are retrieved during
the build process. Whilst the `GO111MODULE` entry can be added to the
`slack.yml` file as per the OpenFaaS documentation this does not appear to be
used when performing shrinkwrap builds, the argument must still be provided when
running `docker buildx build`.

```sh
$ docker buildx build \
 --build-arg GO111MODULE=on \
 --push \
 --tag registry.mydomain.io/openfaas/minio-notification-webhook:latest \
 --platform=linux/amd64,linux/arm64 \
 build/minio-notification-webhook/
```

## Deploying the Function

Run the below commands to point to the OpenFaaS gateway and authenticate.

```sh
$ export OPENFAAS_URL=https://gateway.mydomain.io
$ export PASSWORD=$(kubectl get secret -n openfaas basic-auth -o jsonpath="{.data.basic-auth-password}" | base64 --decode; echo)

$ echo -n $PASSWORD | faas-cli login --username admin --password-stdin
Calling the OpenFaaS server to validate the credentials...
credentials saved for admin https://gateway.mydomain.io
```

```sh
$ faas-cli deploy \
  --image registry.mydomain.io/openfaas/minio-notification-webhook:latest \
  --name minio-notification-webhook \
  --env MINIO_DEBUG=true \
  --env MINIO_LOGLEVEL=debug \
  --env MINIO_SLACK_ENDPOINT=https://gateway.mydomain.io/function/slack

Deployed. 202 Accepted.
URL: https://gateway.mydomain.io/function/minio-notification-webhook
```

Run the below command to remove the function.

```sh
faas-cli remove minio-notification-webhook
```

## Subscribing to MinIO S3 Notifications

Start MinIO with the below environment variables to enable a notification
webhook. The `_SLACK` suffix can be named anything but is used to uniquely
identify the configuration for this notifier. This needs to match the arn for
the webhook as well, i.e. `arn:minio:sqs::SLACK:webhook`

| Name                                | Value                                                             |
| ----------------------------------- | ----------------------------------------------------------------- |
| MINIO_NOTIFY_WEBHOOK_ENABLE_SLACK   | on                                                                |
| MINIO_NOTIFY_WEBHOOK_ENDPOINT_SLACK | <https://gateway.mydomain.io/function/minio-notification-webhook> |
| MINIO_NOTIFY_WEBHOOK_QUEUE_LIMIT    | 10                                                                |

Run the below commands to subscribe to events so that MinIO will send
notification events for objects uploaded, modified, retrieved and deleted in the
`projects` S3 bucket.

```sh
$ mc event add myminio/projects arn:minio:sqs::SLACK:webhook --event put
$ mc event add myminio/projects arn:minio:sqs::SLACK:webhook --event delete
$ mc event add myminio/projects arn:minio:sqs::SLACK:webhook --event get

$ mc event list myminio/projects
arn:minio:sqs::SLACK:webhook   s3:ObjectCreated:*   Filter:
arn:minio:sqs::SLACK:webhook   s3:ObjectRemoved:*   Filter:
arn:minio:sqs::SLACK:webhook   s3:ObjectAccessed:*   Filter:
```

Example json payload for upload of object to S3 bucket.

```json
{
  "EventName": "s3:ObjectCreated:Put",
  "Key": "projects/grafana-monitoring-dashboard.json",
  "Records": [
    {
      "eventVersion": "2.0",
      "eventSource": "minio:s3",
      "awsRegion": "",
      "eventTime": "2021-03-11T07:47:11.994Z",
      "eventName": "s3:ObjectCreated:Put",
      "userIdentity": { "principalId": "XXXXXXXXXX" },
      "requestParameters": {
        "accessKey": "XXXXXXXXXXX",
        "region": "",
        "sourceIPAddress": "10.42.0.156"
      },
      "responseElements": {
        "x-amz-request-id": "166B3A2756316F44",
        "x-minio-deployment-id": "cf5765d8-f665-4689-a019-7481c31c367c",
        "x-minio-origin-endpoint": "http://10.42.0.184:9000"
      },
      "s3": {
        "s3SchemaVersion": "1.0",
        "configurationId": "Config",
        "bucket": {
          "name": "projects",
          "ownerIdentity": { "principalId": "XXXXXXXXXX" },
          "arn": "arn:aws:s3:::projects"
        },
        "object": {
          "key": "grafana-monitoring-dashboard.json",
          "size": 35305,
          "eTag": "c9103045dfdda632353ddc99dab4a674",
          "contentType": "application/json",
          "userMetadata": { "content-type": "application/json" },
          "sequencer": "166B3A2757D47132"
        }
      },
      "source": {
        "host": "10.42.0.156",
        "port": "",
        "userAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.72 Safari/537.36"
      }
    }
  ]
}
```

Example json payload for an object deleted from an S3 bucket.

```json
{
  "EventName": "s3:ObjectRemoved:Delete",
  "Key": "projects/grafana-monitoring-dashboard.json",
  "Records": [
    {
      "eventVersion": "2.0",
      "eventSource": "minio:s3",
      "awsRegion": "",
      "eventTime": "2021-03-11T07:48:28.303Z",
      "eventName": "s3:ObjectRemoved:Delete",
      "userIdentity": { "principalId": "XXXXXXXXX" },
      "requestParameters": {
        "accessKey": "XXXXXXXXXXX",
        "region": "",
        "sourceIPAddress": "10.42.0.156"
      },
      "responseElements": {
        "x-amz-request-id": "",
        "x-minio-deployment-id": "cf5765d8-f665-4689-a019-7481c31c367c",
        "x-minio-origin-endpoint": "http://10.42.0.184:9000"
      },
      "s3": {
        "s3SchemaVersion": "1.0",
        "configurationId": "Config",
        "bucket": {
          "name": "projects",
          "ownerIdentity": { "principalId": "XXXXXXXXXXX" },
          "arn": "arn:aws:s3:::projects"
        },
        "object": {
          "key": "grafana-monitoring-dashboard.json",
          "sequencer": "166B3A391C3CFD09"
        }
      },
      "source": {
        "host": "10.42.0.156",
        "port": "",
        "userAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.72 Safari/537.36"
      }
    }
  ]
}
```

## Credits

Credits to Matt Holt for the [json-to-go] online tool for making my life easy to
generate Go type definitions from JSON.

## License

[![MIT license]](https://lbesson.mit-license.org/)

[arkade]: https://github.com/alexellis/arkade
[docker buildx]:
  https://docs.docker.com/engine/reference/commandline/buildx_build/
[go - dependencies]: https://docs.openfaas.com/cli/templates/#go-go-dependencies
[go modules]: https://golang.org/ref/mod
[json-to-go]: https://mholt.github.io/json-to-go/
[minio]: https://min.io/
[minio bucket notification guide]:
  https://docs.min.io/docs/minio-bucket-notification-guide.html
[mit license]: https://img.shields.io/badge/License-MIT-blue.svg
[openfaas]: https://www.openfaas.com/
[openfaas deployment]: https://docs.openfaas.com/deployment/
[openfaas slack function]: https://github.com/sleighzy/openfaas-slack
[openfaas using secrets]: https://docs.openfaas.com/reference/secrets/
[simple serverless with golang functions and microservices]:
  https://www.openfaas.com/blog/golang-serverless/
[slack-go]: https://github.com/slack-go/slack
[use a private registry with kubernetes]:
  https://docs.openfaas.com/deployment/kubernetes/#use-a-private-registry-with-kubernetes
