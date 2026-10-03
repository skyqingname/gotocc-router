//go:build embed

package web

import (
	"context"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/resellersite"
)

func (s *FrontendServer) cacheForSite(ctx context.Context) *HTMLCache {
	origin := resellersite.CustomerOrigin(ctx)
	if origin == "" {
		return s.cache
	}
	if cached, ok := s.siteCaches.Load(origin); ok {
		return cached.(*HTMLCache)
	}
	cache := NewHTMLCache()
	cache.SetBaseHTML(s.baseHTML)
	actual, _ := s.siteCaches.LoadOrStore(origin, cache)
	return actual.(*HTMLCache)
}
