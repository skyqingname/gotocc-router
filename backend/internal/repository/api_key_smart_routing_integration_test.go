//go:build integration

package repository

import "github.com/LuckyKuang/sub2api-plus/internal/service"

func (s *GatewayRoutingSuite) TestAutoRouteCatalogKeepsGroupIsolation() {
	allowed := mustCreateGroup(s.T(), s.client, &service.Group{Name: "auto-allowed", Platform: service.PlatformComposite, Status: service.StatusActive})
	other := mustCreateGroup(s.T(), s.client, &service.Group{Name: "auto-other", Platform: service.PlatformOpenAI, Status: service.StatusActive})
	for _, item := range []struct {
		name    string
		groupID int64
		status  string
	}{
		{"allowed-account", allowed.ID, service.StatusActive},
		{"other-account", other.ID, service.StatusActive},
		{"disabled-account", allowed.ID, service.StatusDisabled},
	} {
		account := mustCreateAccount(s.T(), s.client, &service.Account{
			Name: item.name, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
			Status: item.status, Schedulable: true,
		})
		mustBindAccountToGroup(s.T(), s.client, account.ID, item.groupID, 1)
	}
	routes := NewCompositeModelRouteRepository(s.client)
	for _, enabled := range []bool{true, false} {
		model := "active-alias"
		if !enabled {
			model = "disabled-alias"
		}
		s.Require().NoError(routes.Create(s.ctx, &service.CompositeModelRoute{
			GroupID: allowed.ID, PublicModel: model, UpstreamModel: "gpt-test",
			TargetPlatform: service.PlatformOpenAI, MatchType: service.CompositeRouteMatchExact,
			Endpoint: service.CompositeRouteEndpointAny, Enabled: enabled,
		}))
	}
	repo := NewAutoRouteCatalogRepository(s.client)
	catalog, err := repo.LoadAutoRouteCatalog(s.ctx, []int64{allowed.ID})
	s.Require().NoError(err)
	s.Require().Len(catalog.Accounts[allowed.ID], 1)
	s.Require().Equal("allowed-account", catalog.Accounts[allowed.ID][0].Name)
	s.Require().NotContains(catalog.Accounts, other.ID)
	s.Require().Len(catalog.Routes[allowed.ID], 1)
	s.Require().Equal("active-alias", catalog.Routes[allowed.ID][0].PublicModel)
	empty, err := repo.LoadAutoRouteCatalog(s.ctx, nil)
	s.Require().NoError(err)
	s.Require().Empty(empty.Accounts)
}
