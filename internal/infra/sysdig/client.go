package sysdig

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

type contextKey string

const (
	contextKeyToken contextKey = "sysdigApiToken"
	contextKeyHost  contextKey = "sysdigApiHost"
)

func WrapContextWithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, contextKeyToken, token)
}

func GetTokenFromContext(ctx context.Context) string {
	return ctx.Value(contextKeyToken).(string)
}

func WrapContextWithHost(ctx context.Context, host string) context.Context {
	return context.WithValue(ctx, contextKeyHost, host)
}

func GetHostFromContext(ctx context.Context) string {
	return ctx.Value(contextKeyHost).(string)
}

func updateReqWithHostURL(req *http.Request, host string) error {
	u, err := url.Parse(host)
	if err != nil {
		// If it's just a hostname without scheme, try prepending https://
		u, err = url.Parse("https://" + host)
		if err != nil {
			return err
		}
	}
	req.URL.Scheme = u.Scheme
	req.URL.Host = u.Host
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

func WithHostAndTokenFromContext() RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		if host, ok := ctx.Value(contextKeyHost).(string); ok && host != "" {
			if err := updateReqWithHostURL(req, host); err != nil {
				return err
			}
		}
		if token, ok := ctx.Value(contextKeyToken).(string); ok && token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
			return nil
		}
		return errors.New("authorization token not present in context")
	}
}

func WithFallbackAuthentication(auths ...RequestEditorFn) RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		for _, auth := range auths {
			if err := auth(ctx, req); err == nil {
				return nil
			}
		}
		return errors.New("unable to authenticate with any method")
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
