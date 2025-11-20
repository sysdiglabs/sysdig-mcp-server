package sysdig

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/oapi-codegen/runtime"
)

// GetProcessTreeBranchesResponse defines model for GetProcessTreeBranches response.
type GetProcessTreeBranchesResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *map[string]any
}

// ParseGetProcessTreeBranchesResponse parses an HTTP response from a GetProcessTreeBranches call
func ParseGetProcessTreeBranchesResponse(rsp *http.Response) (*GetProcessTreeBranchesResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetProcessTreeBranchesResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch rsp.StatusCode {
	case http.StatusOK:
		var dest map[string]any
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest
	case http.StatusNotFound:
		return nil, ErrNotFound
	}
	return response, nil
}

// GetProcessTreeTreesResponse defines model for GetProcessTreeTrees response.
type GetProcessTreeTreesResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *map[string]any
}

// ParseGetProcessTreeTreesResponse parses an HTTP response from a GetProcessTreeTrees call
func ParseGetProcessTreeTreesResponse(rsp *http.Response) (*GetProcessTreeTreesResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetProcessTreeTreesResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch rsp.StatusCode {
	case http.StatusOK:
		var dest map[string]any
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest
	case http.StatusNotFound:
		return nil, ErrNotFound
	}
	return response, nil
}

// NewGetProcessTreeBranchesRequest generates requests for GetProcessTreeBranches
func NewGetProcessTreeBranchesRequest(server string, eventID string) (*http.Request, error) {
	var err error

	var pathParam0 string
	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "eventId", runtime.ParamLocationPath, eventID)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/api/process-tree/v1/process-branches/%s", pathParam0)
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

// NewGetProcessTreeTreesRequest generates requests for GetProcessTreeTrees
func NewGetProcessTreeTreesRequest(server string, eventID string) (*http.Request, error) {
	var err error

	var pathParam0 string
	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "eventId", runtime.ParamLocationPath, eventID)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/api/process-tree/v1/process-trees/%s", pathParam0)
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

func (c *Client) GetProcessTreeBranches(ctx context.Context, eventID string, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetProcessTreeBranchesRequest(c.Server, eventID)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetProcessTreeTrees(ctx context.Context, eventID string, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetProcessTreeTreesRequest(c.Server, eventID)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *ClientWithResponses) GetProcessTreeBranchesWithResponse(ctx context.Context, eventID string, reqEditors ...RequestEditorFn) (*GetProcessTreeBranchesResponse, error) {
	rsp, err := c.ClientInterface.(*Client).GetProcessTreeBranches(ctx, eventID, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetProcessTreeBranchesResponse(rsp)
}

func (c *ClientWithResponses) GetProcessTreeTreesWithResponse(ctx context.Context, eventID string, reqEditors ...RequestEditorFn) (*GetProcessTreeTreesResponse, error) {
	rsp, err := c.ClientInterface.(*Client).GetProcessTreeTrees(ctx, eventID, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetProcessTreeTreesResponse(rsp)
}

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
