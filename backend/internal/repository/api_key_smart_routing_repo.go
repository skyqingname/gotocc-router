package repository

import (
	"context"

	dbent "github.com/LuckyKuang/sub2api-plus/ent"
	"github.com/LuckyKuang/sub2api-plus/ent/account"
	"github.com/LuckyKuang/sub2api-plus/ent/compositemodelroute"
	"github.com/LuckyKuang/sub2api-plus/ent/group"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

type autoRouteCatalogRepository struct {
	client *dbent.Client
}

func NewAutoRouteCatalogRepository(client *dbent.Client) service.AutoRouteCatalogRepository {
	return &autoRouteCatalogRepository{client: client}
}

func (r *autoRouteCatalogRepository) LoadAutoRouteCatalog(ctx context.Context, groupIDs []int64) (*service.AutoRouteCatalog, error) {
	out := &service.AutoRouteCatalog{
		Accounts: make(map[int64][]service.Account),
		Routes:   make(map[int64][]service.CompositeModelRoute),
	}
	client := clientFromContext(ctx, r.client)
	const chunkSize = 256
	for start := 0; start < len(groupIDs); start += chunkSize {
		end := min(start+chunkSize, len(groupIDs))
		ids := groupIDs[start:end]
		accounts, err := client.Account.Query().Where(
			account.DeletedAtIsNil(), account.StatusEQ(service.StatusActive),
			account.SchedulableEQ(true), account.HasGroupsWith(group.IDIn(ids...)),
		).WithGroups(func(q *dbent.GroupQuery) {
			q.Where(group.IDIn(ids...))
		}).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, row := range accounts {
			for _, attached := range row.Edges.Groups {
				out.Accounts[attached.ID] = append(out.Accounts[attached.ID], *accountEntityToService(row))
			}
		}
		routes, err := client.CompositeModelRoute.Query().Where(
			compositemodelroute.GroupIDIn(ids...), compositemodelroute.EnabledEQ(true),
		).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, row := range routes {
			out.Routes[row.GroupID] = append(out.Routes[row.GroupID], *compositeModelRouteEntityToService(row))
		}
	}
	return out, nil
}
