package sysdig

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/oapi-codegen/runtime"
)

// GetEventFieldValuesParams models the query parameters for
// GET /secure/events/v2/eventFields/{field}. The field name is part of the
// URL path, not a query parameter.
type GetEventFieldValuesParams struct {
	Field  string
	From   int64
	To     int64
	Filter *string
}

// GetEventFieldValuesResponse is the typed response from
// GET /secure/events/v2/eventFields/{field}. The body has the shape
// {data: [{label: "suggested", options: [...]}, {label: "other", options: [...]}]}
// where "suggested" values are observed in the window and "other" values
// are known in the tenant but inactive in the window.
type GetEventFieldValuesResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *map[string]any
}

func ParseGetEventFieldValuesResponse(rsp *http.Response) (*GetEventFieldValuesResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetEventFieldValuesResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	if rsp.StatusCode == http.StatusOK {
		var dest map[string]any
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest
	}
	return response, nil
}

func (r *GetEventFieldValuesResponse) StatusCode() int { return r.HTTPResponse.StatusCode }

func NewGetEventFieldValuesRequest(server string, params *GetEventFieldValuesParams) (*http.Request, error) {
	if params.Field == "" {
		return nil, fmt.Errorf("field is required")
	}

	pathParam, err := runtime.StyleParamWithLocation("simple", false, "field", runtime.ParamLocationPath, params.Field)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	queryURL, err := serverURL.Parse(fmt.Sprintf("./secure/events/v2/eventFields/%s", pathParam))
	if err != nil {
		return nil, err
	}

	q := queryURL.Query()
	q.Set("from", strconv.FormatInt(params.From, 10))
	q.Set("to", strconv.FormatInt(params.To, 10))
	if params.Filter != nil {
		q.Set("filter", *params.Filter)
	}
	queryURL.RawQuery = q.Encode()

	return http.NewRequest("GET", queryURL.String(), nil)
}

func (c *Client) GetEventFieldValues(ctx context.Context, params *GetEventFieldValuesParams, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetEventFieldValuesRequest(c.Server, params)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *ClientWithResponses) GetEventFieldValuesWithResponse(ctx context.Context, params *GetEventFieldValuesParams, reqEditors ...RequestEditorFn) (*GetEventFieldValuesResponse, error) {
	rsp, err := c.ClientInterface.(*Client).GetEventFieldValues(ctx, params, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetEventFieldValuesResponse(rsp)
}
