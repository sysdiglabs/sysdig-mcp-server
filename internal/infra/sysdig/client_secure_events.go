package sysdig

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// GetSecureEventsCountParams models the query parameters for
// GET /api/v1/secureEvents/count.
type GetSecureEventsCountParams struct {
	From   int64
	To     int64
	Filter *string
}

// GetSecureEventsCountResponse is the typed response from
// GET /api/v1/secureEvents/count. The endpoint returns 16 event categories
// (policyEvents, scanningEvents, cloudTrailEvents, etc.), each with a
// severity histogram keyed "0"-"7".
type GetSecureEventsCountResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *map[string]any
}

func ParseGetSecureEventsCountResponse(rsp *http.Response) (*GetSecureEventsCountResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetSecureEventsCountResponse{
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

func (r *GetSecureEventsCountResponse) StatusCode() int { return r.HTTPResponse.StatusCode }

func NewGetSecureEventsCountRequest(server string, params *GetSecureEventsCountParams) (*http.Request, error) {
	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	queryURL, err := serverURL.Parse("./api/v1/secureEvents/count")
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

// GetSecureEventsTimeseriesByParams models the query parameters for
// GET /api/v1/secureEvents/timeseriesBy.
type GetSecureEventsTimeseriesByParams struct {
	From   int64
	To     int64
	Field  string // categorical field to group by (e.g. "severity")
	Rows   int32  // upper bound on bucket count; server picks the coarsest step <= rows
	Limit  int32  // max distinct values reported under Field; server requires it
	Filter *string
}

// GetSecureEventsTimeseriesByResponse is the typed response from
// GET /api/v1/secureEvents/timeseriesBy. Returns a nested subCount tree
// indexed by group-value -> "timestamp" -> bucket-ns -> count, plus a
// top-level `step` (bucket width in nanoseconds).
type GetSecureEventsTimeseriesByResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *map[string]any
}

func ParseGetSecureEventsTimeseriesByResponse(rsp *http.Response) (*GetSecureEventsTimeseriesByResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetSecureEventsTimeseriesByResponse{
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

func (r *GetSecureEventsTimeseriesByResponse) StatusCode() int { return r.HTTPResponse.StatusCode }

func NewGetSecureEventsTimeseriesByRequest(server string, params *GetSecureEventsTimeseriesByParams) (*http.Request, error) {
	if params.Field == "" {
		return nil, fmt.Errorf("field is required")
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	queryURL, err := serverURL.Parse("./api/v1/secureEvents/timeseriesBy")
	if err != nil {
		return nil, err
	}

	q := queryURL.Query()
	q.Set("from", strconv.FormatInt(params.From, 10))
	q.Set("to", strconv.FormatInt(params.To, 10))
	q.Set("field", params.Field)
	q.Set("rows", strconv.FormatInt(int64(params.Rows), 10))
	q.Set("limit", strconv.FormatInt(int64(params.Limit), 10))
	if params.Filter != nil {
		q.Set("filter", *params.Filter)
	}
	queryURL.RawQuery = q.Encode()

	return http.NewRequest("GET", queryURL.String(), nil)
}

func (c *Client) GetSecureEventsCount(ctx context.Context, params *GetSecureEventsCountParams, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetSecureEventsCountRequest(c.Server, params)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetSecureEventsTimeseriesBy(ctx context.Context, params *GetSecureEventsTimeseriesByParams, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetSecureEventsTimeseriesByRequest(c.Server, params)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *ClientWithResponses) GetSecureEventsCountWithResponse(ctx context.Context, params *GetSecureEventsCountParams, reqEditors ...RequestEditorFn) (*GetSecureEventsCountResponse, error) {
	rsp, err := c.ClientInterface.(*Client).GetSecureEventsCount(ctx, params, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetSecureEventsCountResponse(rsp)
}

func (c *ClientWithResponses) GetSecureEventsTimeseriesByWithResponse(ctx context.Context, params *GetSecureEventsTimeseriesByParams, reqEditors ...RequestEditorFn) (*GetSecureEventsTimeseriesByResponse, error) {
	rsp, err := c.ClientInterface.(*Client).GetSecureEventsTimeseriesBy(ctx, params, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetSecureEventsTimeseriesByResponse(rsp)
}
