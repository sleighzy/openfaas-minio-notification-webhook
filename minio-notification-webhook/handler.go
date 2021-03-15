package function

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	handler "github.com/openfaas/templates-sdk/go-http"

	log "github.com/sirupsen/logrus"
)

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

	var err error

	debugEnv, _ := os.LookupEnv("MINIO_DEBUG")
	debug, err := strconv.ParseBool(debugEnv)
	if err != nil {
		log.Fatal(err)
	}

	message := fmt.Sprintf("Body: %s", string(req.Body))

	return handler.Response{
		Body:       []byte(message),
		StatusCode: http.StatusOK,
	}, err
}
