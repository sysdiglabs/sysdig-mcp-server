package sysdig

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

// UserPermissions defines model for user permissions.
type UserPermissions struct {
	CustomerID  int      `json:"customerId"`
	UserID      int      `json:"userId"`
	TeamID      int      `json:"teamId"`
	Permissions []string `json:"permissions"`
}

// GetMyPermissionsResponse defines model for GetMyPermissions response.
type GetMyPermissionsResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *UserPermissions
}

// ParseGetMyPermissionsResponse parses an HTTP response from a GetMyPermissions call
func ParseGetMyPermissionsResponse(rsp *http.Response) (*GetMyPermissionsResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetMyPermissionsResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch rsp.StatusCode {
	case http.StatusOK:
		var dest UserPermissions
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest
	}
	return response, nil
}

// NewGetMyPermissionsRequest generates requests for GetMyPermissions
func NewGetMyPermissionsRequest(server string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := "/api/users/me/permissions"
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (c *Client) GetMyPermissions(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetMyPermissionsRequest(c.Server)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *ClientWithResponses) GetMyPermissionsWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetMyPermissionsResponse, error) {
	rsp, err := c.ClientInterface.(*Client).GetMyPermissions(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetMyPermissionsResponse(rsp)
}
