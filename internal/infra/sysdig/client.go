package sysdig

import (
	"context"
	"errors"
	"net/http"
)

func NewSysdigClient(apiEndpoint, apiToken string) (ExtendedClientWithResponsesInterface, error) {
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
