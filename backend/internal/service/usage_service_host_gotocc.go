package service

import (
	"context"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/resellersite"
)

// usageServiceHost records which public entry point (gotocc.xyz, api.gotocc.xyz,
// a customer domain) received the request. Detached work has no entry host.
func usageServiceHost(ctx context.Context) *string {
	host := resellersite.EntryHost(ctx)
	if host == "" {
		return nil
	}
	return &host
}
