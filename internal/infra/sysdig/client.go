package sysdig

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func updateReqWithHostURL(req *http.Request, host string) error {
	u, err := url.Parse(host)
	if err != nil {
		return err
	}
	if !u.IsAbs() || u.Host == "" {
		return fmt.Errorf("Sysdig API host must be an absolute URL")
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("Sysdig API host must not contain user information, a query string, or a fragment")
	}

	req.URL.Scheme = u.Scheme
	req.URL.Host = u.Host

	basePath := strings.TrimSuffix(u.Path, "/")
	if basePath != "" {
		req.URL.Path = basePath + "/" + strings.TrimPrefix(req.URL.Path, "/")
		req.URL.RawPath = ""
	}
	return nil
}

func WithFixedHostAndToken(host, apiToken string) RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		if err := updateReqWithHostURL(req, host); err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+apiToken)
		return nil
	}
}

func WithVersion(version string) RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("User-Agent", fmt.Sprintf("sysdig-mcp-server/%s", version))
		return nil
	}
}

type IntoClientOption interface {
	AsClientOption() ClientOption
}

func (r RequestEditorFn) AsClientOption() ClientOption {
	return WithRequestEditorFn(r)
}

func (c ClientOption) AsClientOption() ClientOption {
	return c
}

func NewSysdigClient(requestEditors ...IntoClientOption) (ExtendedClientWithResponsesInterface, error) {
	editors := make([]ClientOption, len(requestEditors))
	for i, e := range requestEditors {
		editors[i] = e.AsClientOption()
	}

	return NewClientWithResponses("", editors...)
}
