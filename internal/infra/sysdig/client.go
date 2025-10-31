package sysdig

//go:generate mockgen -source=client.gen.go -destination=./mocks/${GOFILE} -package=mocks

import (
	"context"
	"errors"
	"net/http"
)

func NewSysdigClient(apiEndpoint, apiToken string) (ClientWithResponsesInterface, error) {
	if apiEndpoint == "" {
		return nil, errors.New("the api endpoint is empty")
	}
	if apiToken == "" {
		return nil, errors.New("the api token is empty")
	}

	return NewClientWithResponses(apiEndpoint, WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+apiToken)
		return nil
	}))
}
