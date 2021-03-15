package function

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	handler "github.com/openfaas/templates-sdk/go-http"

	log "github.com/sirupsen/logrus"
)

type NotificationEvent struct {
	EventName string `json:"EventName"`
	Key       string `json:"Key"`

	Records []struct {
		EventVersion string    `json:"eventVersion"`
		EventSource  string    `json:"eventSource"`
		AwsRegion    string    `json:"awsRegion"`
		EventName    string    `json:"eventName"`
		EventTime    time.Time `json:"eventTime"`

		UserIdentity struct {
			PrincipalId string `json:"principalId"`
		} `json:"userIdentity"`

		RequestParameters struct {
			AccessKey       string `json:"accessKey"`
			Region          string `json:"region"`
			SourceIPAddress string `json:"sourceIPAddress"`
		} `json:"requestParameters"`

		ResponseElements struct {
			XAmzRequestId        string `json:"x-amz-request-id"`
			XMinioDeploymentId   string `json:"x-minio-deployment-id"`
			XMinioOriginEndpoint string `json:"x-minio-origin-endpoint"`
		} `json:"responseElements"`

		S3 struct {
			S3SchemaVersion string `json:"s3SchemaVersion"`
			ConfigurationId string `json:"configurationId"`

			Bucket struct {
				Arn  string `json:"arn"`
				Name string `json:"name"`

				OwnerIdentity struct {
					PrincipalId string `json:"principalId"`
				} `json:"ownerIdentity"`
			} `json:"bucket"`

			Object struct {
				Size        int    `json:"size"`
				Key         string `json:"key"`
				ETag        string `json:"eTag"`
				ContentType string `json:"contentType"`
				Sequencer   string `json:"sequencer"`

				UserMetadata struct {
					ContentType string `json:"content-type"`
				} `json:"userMetadata"`
			} `json:"object"`
		} `json:"s3"`

		Source struct {
			Host      string `json:"host"`
			Port      string `json:"port"`
			UserAgent string `json:"userAgent"`
		} `json:"source"`
	} `json:"Records"`
}

type Message struct {
	Title string
	Body  Body
}

type Body struct {
	Text string
}

// Handle a function invocation
func Handle(req handler.Request) (handler.Response, error) {
	if loglevel, ok := os.LookupEnv("MINIO_LOGLEVEL"); ok {
		switch loglevel {
		case "debug":
			log.SetLevel(log.DebugLevel)
		case "warn":
			log.SetLevel(log.WarnLevel)
		case "error":
			log.SetLevel(log.ErrorLevel)
		case "fatal":
			log.SetLevel(log.FatalLevel)
		default:
			log.SetLevel(log.InfoLevel)
		}
	}

	log.Debug(fmt.Sprintf("Event: %s", string(req.Body)))

	var err error

	endpoint, exists := os.LookupEnv("MINIO_SLACK_ENDPOINT")
	if !exists || len(endpoint) <= 0 {
		log.Error("The MINIO_SLACK_ENDPOINT value has not been set, events will not be sent to Slack")

		return handler.Response{
			Body:       []byte("The MINIO_SLACK_ENDPOINT value has not been set"),
			StatusCode: http.StatusInternalServerError,
		}, err
	}

	var event NotificationEvent

	err = json.Unmarshal(req.Body, &event)
	if err != nil {
		log.Fatal(err)
	}

	log.Debug(fmt.Sprintf("EventName: '%s'", event.EventName))

	messageBody := Body{Text: fmt.Sprintf("Bucket: '%s'\nObject: '%s'\nAction: '%s'\nPrincipal: '%s'",
		event.Records[0].S3.Bucket.Name, event.Records[0].S3.Object.Key, event.EventName, event.Records[0].S3.Bucket.OwnerIdentity.PrincipalId)}
	messageText := Message{Title: event.EventName, Body: messageBody}
	message, err := json.Marshal(messageText)
	if err != nil {
		log.Fatal(err)
	}

	log.Debug(fmt.Sprintf("Message: '%s'", message))

	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(message))
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return handler.Response{
		Body:       []byte(body),
		StatusCode: http.StatusOK,
	}, err
}
