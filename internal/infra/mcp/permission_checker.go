package mcp

import (
	"context"
	"slices"
	"sync"

	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

type PermissionChecker interface {
	HasPermission(permission string) bool
}

type CachedPermissionChecker struct {
	sysdigClient sysdig.ExtendedClientWithResponsesInterface
	once         sync.Once
	permissions  []string
	err          error
}

func NewPermissionChecker(client sysdig.ExtendedClientWithResponsesInterface) PermissionChecker {
	return &CachedPermissionChecker{
		sysdigClient: client,
	}
}

func (c *CachedPermissionChecker) loadPermissions() {
	resp, err := c.sysdigClient.GetMyPermissionsWithResponse(context.Background())
	if err != nil {
		c.err = err
		return
	}
	if resp.JSON200 != nil {
		c.permissions = resp.JSON200.Permissions
	}
}

func (c *CachedPermissionChecker) HasPermission(permission string) bool {
	c.once.Do(c.loadPermissions)
	if c.err != nil {
		return false
	}
	return slices.Contains(c.permissions, permission)
}
