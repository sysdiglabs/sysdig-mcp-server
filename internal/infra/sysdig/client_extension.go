package sysdig

import "context"

//go:generate mockgen -source=$GOFILE -destination=mocks/$GOFILE -package=mocks

type ExtendedClientWithResponsesInterface interface {
	ClientInterface
	ClientWithResponsesInterface
	GetProcessTreeBranchesWithResponse(ctx context.Context, eventID string, reqEditors ...RequestEditorFn) (*GetProcessTreeBranchesResponse, error)
	GetProcessTreeTreesWithResponse(ctx context.Context, eventID string, reqEditors ...RequestEditorFn) (*GetProcessTreeTreesResponse, error)
	GetMyPermissionsWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetMyPermissionsResponse, error)
	GenerateSysqlWithResponse(ctx context.Context, question string, reqEditors ...RequestEditorFn) (*GenerateSysqlResponse, error)
}
