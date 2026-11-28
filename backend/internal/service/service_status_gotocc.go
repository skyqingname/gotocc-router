package service

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/timezone"
	"golang.org/x/sync/singleflight"
)

// 服务状态：只看真实用户请求的结果，不主动探测。每个分组一张卡：最近 N 次请求的成功率定
// 当前状态，近 24 小时每小时一格。成功是请求最终正常完成（中途换号后成功也算）；失败是
// 平台或上游原因（上游报错、超时、没有可用账号等）；用户自己的原因（参数、鉴权、余额、
// 内容审核、客户端断开）既不算失败也不计入请求。阈值、窗口、条数和缓存时长来自根目录
// service-status.json（tools/generate_service_status.py 生成 service_status_config_gotocc.go）。

type ServiceStatusConfig struct {
	RecentRequests      int
	NormalSuccessRate   float64
	DegradedSuccessRate float64
	BucketCount         int
	BucketMinutes       int
	CacheSeconds        int
}

const (
	ServiceStatusNormal   = "normal"
	ServiceStatusDegraded = "degraded"
	ServiceStatusOutage   = "outage"
	ServiceStatusIdle     = "idle"
)

// ServiceStatusCounts 是一个分组一个时间格里的请求数。
type ServiceStatusCounts struct {
	Total   int `json:"total"`
	Success int `json:"success"`
}

// ServiceStatusGroupStats 是一个分组在整个窗口里的原始统计，由仓储按全站一次算出。
type ServiceStatusGroupStats struct {
	Buckets          []ServiceStatusCounts
	Recent           ServiceStatusCounts
	TTFTP50Ms        *float64
	InputTokens      int64
	CacheReadTokens  int64
	CacheWriteTokens int64
}

type ServiceStatusRepository interface {
	Read(ctx context.Context, start, end time.Time, bucket time.Duration, buckets, recent int) (map[int64]*ServiceStatusGroupStats, error)
}

type ServiceStatusBucket struct {
	Start   time.Time `json:"start"`
	End     time.Time `json:"end"`
	Total   int       `json:"total"`
	Success int       `json:"success"`
	Status  string    `json:"status"`
}

type ServiceStatusGroup struct {
	ID               int64                 `json:"id"`
	Name             string                `json:"name"`
	Description      string                `json:"description"`
	Platform         string                `json:"platform"`
	SubscriptionType string                `json:"subscription_type"`
	IsExclusive      bool                  `json:"is_exclusive"`
	RateMultiplier   float64               `json:"rate_multiplier"`
	Status           string                `json:"status"`
	Recent           ServiceStatusCounts   `json:"recent"`
	Day              ServiceStatusCounts   `json:"day"`
	TTFTP50Ms        *float64              `json:"ttft_p50_ms"`
	CacheHitRate     *float64              `json:"cache_hit_rate"`
	Buckets          []ServiceStatusBucket `json:"buckets"`
}

type ServiceStatusSnapshot struct {
	GeneratedAt    time.Time            `json:"generated_at"`
	Timezone       string               `json:"timezone"`
	RecentRequests int                  `json:"recent_requests"`
	BucketMinutes  int                  `json:"bucket_minutes"`
	Groups         []ServiceStatusGroup `json:"groups"`
}

type ServiceStatusModel struct {
	Name     string `json:"name"`
	Platform string `json:"platform"`
}

type serviceStatusWindow struct {
	at      time.Time
	start   time.Time
	buckets []ServiceStatusBucket
	stats   map[int64]*ServiceStatusGroupStats
}

type serviceStatusGroupReader interface {
	GetAvailableGroupsForScope(context.Context, int64, string) ([]Group, error)
	GetUserGroupRatesForScope(context.Context, int64, string) (map[int64]float64, error)
}

type ServiceStatusService struct {
	repo    ServiceStatusRepository
	groups  serviceStatusGroupReader
	catalog *ModelPlazaService
	mu      sync.Mutex
	cached  *serviceStatusWindow
	load    singleflight.Group
}

func NewServiceStatusService(repo ServiceStatusRepository, apiKeys *APIKeyService, catalog *ModelPlazaService) *ServiceStatusService {
	return &ServiceStatusService{repo: repo, groups: apiKeys, catalog: catalog}
}

func serviceStatusOf(counts ServiceStatusCounts) string {
	if counts.Total == 0 {
		return ServiceStatusIdle
	}
	rate := float64(counts.Success) / float64(counts.Total)
	switch {
	case rate >= serviceStatusConfig.NormalSuccessRate:
		return ServiceStatusNormal
	case rate >= serviceStatusConfig.DegradedSuccessRate:
		return ServiceStatusDegraded
	default:
		return ServiceStatusOutage
	}
}

// window 返回全站共用的统计，按 CacheSeconds 缓存；同一时刻多人打开只查一次库。
func (s *ServiceStatusService) window(ctx context.Context) (*serviceStatusWindow, error) {
	ttl := time.Duration(serviceStatusConfig.CacheSeconds) * time.Second
	s.mu.Lock()
	cached := s.cached
	s.mu.Unlock()
	if cached != nil && time.Since(cached.at) < ttl {
		return cached, nil
	}
	value, err, _ := s.load.Do("window", func() (any, error) {
		now := timezone.Now()
		bucket := time.Duration(serviceStatusConfig.BucketMinutes) * time.Minute
		// 最后一格是当前这一小时（按站点时区对齐），往前共 BucketCount 格。
		current := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
		current = current.Add(now.Sub(current).Truncate(bucket))
		start := current.Add(-bucket * time.Duration(serviceStatusConfig.BucketCount-1))
		stats, err := s.repo.Read(ctx, start, now, bucket, serviceStatusConfig.BucketCount, serviceStatusConfig.RecentRequests)
		if err != nil {
			return nil, err
		}
		buckets := make([]ServiceStatusBucket, serviceStatusConfig.BucketCount)
		for i := range buckets {
			buckets[i].Start = start.Add(bucket * time.Duration(i))
			buckets[i].End = buckets[i].Start.Add(bucket)
		}
		window := &serviceStatusWindow{at: time.Now(), start: start, buckets: buckets, stats: stats}
		s.mu.Lock()
		s.cached = window
		s.mu.Unlock()
		return window, nil
	})
	if err != nil {
		return nil, err
	}
	return value.(*serviceStatusWindow), nil
}

// Snapshot 按管理员排好的顺序列出 userID 能用的分组（与可用渠道同一套可见性规则）及其被动状态。倍率按该用户
// 实际生效的倍率；站长客户拿到的分组已换成站长给他的客户价，不含站长成本。
func (s *ServiceStatusService) Snapshot(ctx context.Context, userID int64, acceptLanguage string) (*ServiceStatusSnapshot, error) {
	groups, err := s.groups.GetAvailableGroupsForScope(ctx, userID, "personal")
	if err != nil {
		return nil, err
	}
	rates, err := s.groups.GetUserGroupRatesForScope(ctx, userID, "personal")
	if err != nil {
		return nil, err
	}
	window, err := s.window(ctx)
	if err != nil {
		return nil, err
	}
	now := timezone.Now()
	out := &ServiceStatusSnapshot{
		GeneratedAt: window.at, Timezone: timezone.Name(),
		RecentRequests: serviceStatusConfig.RecentRequests, BucketMinutes: serviceStatusConfig.BucketMinutes,
		Groups: make([]ServiceStatusGroup, 0, len(groups)),
	}
	for i := range groups {
		group := groups[i]
		var personal *float64
		if rate, ok := rates[group.ID]; ok {
			personal = &rate
		}
		name, description := group.I18n.Localize(acceptLanguage, group.Name, group.Description)
		item := ServiceStatusGroup{
			ID: group.ID, Name: name, Description: description, Platform: group.Platform,
			SubscriptionType: group.SubscriptionType, IsExclusive: group.IsExclusive,
			RateMultiplier: ResolveBillingRateQuote(&group, group.Platform, "", "token", personal, now).EffectiveMultiplier,
			Buckets:        make([]ServiceStatusBucket, len(window.buckets)),
		}
		copy(item.Buckets, window.buckets)
		if stats := window.stats[group.ID]; stats != nil {
			for b, counts := range stats.Buckets {
				item.Buckets[b].Total, item.Buckets[b].Success = counts.Total, counts.Success
				item.Day.Total += counts.Total
				item.Day.Success += counts.Success
			}
			item.Recent = stats.Recent
			item.TTFTP50Ms = stats.TTFTP50Ms
			if inputSide := stats.InputTokens + stats.CacheReadTokens + stats.CacheWriteTokens; inputSide > 0 {
				rate := float64(stats.CacheReadTokens) / float64(inputSide)
				item.CacheHitRate = &rate
			}
		}
		for b := range item.Buckets {
			item.Buckets[b].Status = serviceStatusOf(ServiceStatusCounts{Total: item.Buckets[b].Total, Success: item.Buckets[b].Success})
		}
		item.Status = serviceStatusOf(item.Recent)
		out.Groups = append(out.Groups, item)
	}
	return out, nil
}

// Models 列出该用户能用的某个分组可调用的模型，取自可用渠道同一份目录。
func (s *ServiceStatusService) Models(ctx context.Context, userID, groupID int64) ([]ServiceStatusModel, error) {
	groups, err := s.groups.GetAvailableGroupsForScope(ctx, userID, "personal")
	if err != nil {
		return nil, err
	}
	for _, group := range groups {
		if group.ID != groupID {
			continue
		}
		catalog, err := s.catalog.ListAvailableChannelCatalog(ctx, []Group{group})
		if err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		models := []ServiceStatusModel{}
		for _, entry := range catalog {
			for _, model := range entry.Models {
				key := model.Platform + "\x00" + strings.ToLower(model.Name)
				if !seen[key] {
					seen[key] = true
					models = append(models, ServiceStatusModel{Name: model.Name, Platform: model.Platform})
				}
			}
		}
		sort.Slice(models, func(a, b int) bool {
			if models[a].Platform != models[b].Platform {
				return models[a].Platform < models[b].Platform
			}
			return models[a].Name < models[b].Name
		})
		return models, nil
	}
	return nil, ErrGroupNotFound
}
