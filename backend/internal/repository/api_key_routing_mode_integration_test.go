//go:build integration

package repository

import "github.com/LuckyKuang/sub2api-plus/internal/service"

func (s *APIKeyRepoSuite) TestAutoRoutingPersistsInAuthenticationProjection() {
	user := s.mustCreateUser("auto-routing-auth@example.com")
	key := &service.APIKey{
		UserID: user.ID, Key: "sk-auto-routing-auth", Name: "Auto",
		Status: service.StatusActive, RoutingMode: service.APIKeyRoutingAuto,
	}
	s.Require().NoError(s.repo.Create(s.ctx, key))

	stored, err := s.repo.GetByID(s.ctx, key.ID)
	s.Require().NoError(err)
	s.Require().True(stored.IsAutoRouting())
	s.Require().Nil(stored.GroupID)
	forAuth, err := s.repo.GetByKeyForAuth(s.ctx, key.Key)
	s.Require().NoError(err)
	s.Require().True(forAuth.IsAutoRouting())
	s.Require().Nil(forAuth.GroupID)
}

func (s *APIKeyRepoSuite) TestAutoRoutingUpdatePreservesConcurrentUsage() {
	user := s.mustCreateUser("auto-routing-update@example.com")
	group := s.mustCreateGroup("auto-routing-original")
	key := &service.APIKey{
		UserID: user.ID, Key: "sk-auto-routing-update", Name: "Before",
		Status: service.StatusActive, GroupID: &group.ID, Quota: 100,
	}
	s.Require().NoError(s.repo.Create(s.ctx, key))
	stale, err := s.repo.GetByID(s.ctx, key.ID)
	s.Require().NoError(err)
	_, err = s.repo.IncrementQuotaUsed(s.ctx, key.ID, 7)
	s.Require().NoError(err)
	s.Require().NoError(s.repo.IncrementRateLimitUsage(s.ctx, key.ID, 7))

	stale.RoutingMode = service.APIKeyRoutingAuto
	stale.GroupID = nil
	s.Require().NoError(s.repo.Update(s.ctx, stale, service.APIKeyUpdateFields{
		RoutingMode: true, GroupID: true,
	}))
	stored, err := s.repo.GetByID(s.ctx, key.ID)
	s.Require().NoError(err)
	s.Require().True(stored.IsAutoRouting())
	s.Require().Nil(stored.GroupID)
	s.Require().InDelta(7, stored.QuotaUsed, 1e-9)
	s.Require().InDelta(7, stored.Usage5h, 1e-9)
}

func (s *APIKeyRepoSuite) TestAutoRoutingCannotPersistAFixedGroup() {
	user := s.mustCreateUser("auto-routing-constraint@example.com")
	group := s.mustCreateGroup("auto-routing-constraint")
	key := &service.APIKey{
		UserID: user.ID, Key: "sk-auto-routing-constraint", Name: "Invalid",
		Status: service.StatusActive, GroupID: &group.ID, RoutingMode: service.APIKeyRoutingAuto,
	}
	err := s.repo.Create(s.ctx, key)
	s.Require().Error(err)
}
