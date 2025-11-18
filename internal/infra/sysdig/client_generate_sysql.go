package sysdig

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type GenerateSysqlResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *SysqlQuery
}

func (r *GenerateSysqlResponse) String() string {
	if r.Body != nil {
		return string(r.Body)
	}
	return ""
}

type SysqlQuery struct {
	ID       string `json:"id,omitempty"`
	Text     string `json:"text"`
	Sender   string `json:"sender,omitempty"`
	Type     string `json:"type,omitempty"`
	Time     string `json:"time,omitempty"`
	Status   string `json:"status,omitempty"`
	Metadata *struct {
		SubscriptionInfo struct {
			MonthlyLimit   int `json:"monthly_limit,omitempty"`
			MonthlyCount   int `json:"monthly_count,omitempty"`
			WarningPercent int `json:"warning_percent,omitempty"`
		} `json:"subscription_info"`
		TranslateErrorType   *string  `json:"translate_error_type,omitempty"`
		AlternativeQuestions []string `json:"alternative_questions,omitempty"`
	} `json:"metadata,omitempty"`
}

func NewGenerateSysqlRequest(server string, question string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := "/api/sage/sysql/generate"
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	queryValues := queryURL.Query()
	queryValues.Set("question", question)
	queryURL.RawQuery = queryValues.Encode()

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (c *Client) GenerateSysql(ctx context.Context, question string, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGenerateSysqlRequest(c.Server, question)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func ParseGenerateSysqlResponse(rsp *http.Response) (*GenerateSysqlResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GenerateSysqlResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	temp := &SysqlQuery{}
	if err := json.Unmarshal(bodyBytes, &temp); err != nil {
		return nil, fmt.Errorf("failed to parse generate sysql response: %w. Body: %s", err, string(bodyBytes))
	}

	switch rsp.StatusCode {
	case http.StatusOK:
		var dest SysqlQuery
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest
	}
	return response, nil
}

func (c *ClientWithResponses) GenerateSysqlWithResponse(ctx context.Context, question string, reqEditors ...RequestEditorFn) (*GenerateSysqlResponse, error) {
	rsp, err := c.ClientInterface.(*Client).GenerateSysql(ctx, question, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGenerateSysqlResponse(rsp)
}
