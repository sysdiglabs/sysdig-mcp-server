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
	GetSecureEventsCountWithResponse(ctx context.Context, params *GetSecureEventsCountParams, reqEditors ...RequestEditorFn) (*GetSecureEventsCountResponse, error)
	GetSecureEventsTimeseriesByWithResponse(ctx context.Context, params *GetSecureEventsTimeseriesByParams, reqEditors ...RequestEditorFn) (*GetSecureEventsTimeseriesByResponse, error)
	GetEventFieldValuesWithResponse(ctx context.Context, params *GetEventFieldValuesParams, reqEditors ...RequestEditorFn) (*GetEventFieldValuesResponse, error)
}
