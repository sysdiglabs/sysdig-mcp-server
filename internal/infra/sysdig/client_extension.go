package sysdig

import "context"

//go:generate mockgen -source=$GOFILE -destination=mocks/$GOFILE -package=mocks

type ExtendedClientWithResponsesInterface interface {
	ClientInterface
	ClientWithResponsesInterface
	GetMyPermissionsWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetMyPermissionsResponse, error)
	GenerateSysqlWithResponse(ctx context.Context, question string, reqEditors ...RequestEditorFn) (*GenerateSysqlResponse, error)
}
