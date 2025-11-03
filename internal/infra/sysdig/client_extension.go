package sysdig

import "context"

//go:generate mockgen -source=$GOFILE -destination=mocks/$GOFILE -package=mocks

type ExtendedClientWithResponsesInterface interface {
	ClientInterface
	ClientWithResponsesInterface
	GetProcessTreeBranchesWithResponse(ctx context.Context, eventId string, reqEditors ...RequestEditorFn) (*GetProcessTreeBranchesResponse, error)
	GetProcessTreeTreesWithResponse(ctx context.Context, eventId string, reqEditors ...RequestEditorFn) (*GetProcessTreeTreesResponse, error)
}
